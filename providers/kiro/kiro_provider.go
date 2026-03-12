package kiro

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream"
	"github.com/google/uuid"
	kirocreds "github.com/nomand-zc/lumin-client/credentials/kiro"
	"github.com/nomand-zc/lumin-client/httpclient"
	"github.com/nomand-zc/lumin-client/log"
	"github.com/nomand-zc/lumin-client/providers"
	"github.com/nomand-zc/lumin-client/providers/kiro/converter"
	"github.com/nomand-zc/lumin-client/providers/kiro/converter/parser"
	"github.com/nomand-zc/lumin-client/queue"
)

func init() {
	providers.Register(NewProvider(providers.DefaultProviderName))
}

const (
	providerName     = "kiro"
	defaultQueueSize = 100
	// 事件流解码 payload 缓冲区大小
	defaultPayloadBufSize = 10 * 1024
)

type kiroProvider struct {
	name string
	httpClient httpclient.HTTPClient
	options    *Options
}

// NewProvider creates a new kiro provider.
func NewProvider(name string, opts ...Option) *kiroProvider {
	options := &defaultOptions
	for _, opt := range opts {
		opt(options)
	}
	return &kiroProvider{
		name:     	name,
		options:    options,
		httpClient: httpclient.New(httpclient.WithMiddleware(
			httpclient.LoggingMiddleware,
		)),
	}
}

// Name returns the name of the provider.
func (p *kiroProvider) Name() string {
	return p.name
}

// Type returns the type of the provider.
func (p *kiroProvider) Type() string {
	return providerName
}

// GenerateContent generates content.
func (p *kiroProvider) GenerateContent(ctx context.Context, 
	req *providers.Request) (*providers.Response, error) {
	reader, err := p.GenerateContentStream(ctx, req)
	if err != nil {
		return nil, err
	}

	acc := &providers.ResponseAccumulator{}
	if err := reader.Each(ctx, func(chunk *providers.Response) error {
		if !acc.AddChunk(chunk) {
			log.Warnf("[kiroProvider.GenerateContent] failed to accumulate chunk, id mismatch")
			
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("failed to read stream response: %w", err)
	}

	resp := acc.Response()
	if resp == nil {
		return nil, fmt.Errorf("no response received from stream")
	}
	// 将 Object 标记为非流式响应类型
	resp.Object = providers.ObjectChatCompletion
	resp.IsPartial = false
	resp.Done = true
	return resp, nil
}

// GenerateContentStream generates content in a stream.
func (p *kiroProvider) GenerateContentStream(ctx context.Context, 
	req *providers.Request) (queue.Consumer[*providers.Response], error){
	// 1. 初始化调用上下文
	ctx, inv := providers.EnsureInvocationContext(ctx)
	inputTokens, err := p.options.tokenConter.CountTokensRange(ctx, req.Messages, 0, len(req.Messages))
	if err != nil {
		// token 计算失败
		return nil, fmt.Errorf("failed to calculate tokens: %w", err)
	}
	inv.Usage.PromptTokens = inputTokens
	inv.ID = uuid.NewString()

	// 2. 构建请求信息
	kiroCreds := req.Credential.(*kirocreds.Credential)
	url := fmt.Sprintf(p.options.url, kiroCreds.Region)
	cwReq, err := converter.ConvertRequest(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to convert request: %w", err)
	}
	if cwReq == nil {
		return nil, fmt.Errorf("request was filtered out during conversion")
	}
	if kiroCreds.AuthMethod == kirocreds.AuthMethodSocial && kiroCreds.ProfileArn != "" {
		cwReq.ProfileArn = kiroCreds.ProfileArn
	}
	cwReqBody, err := json.Marshal(cwReq)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(cwReqBody))
	if err != nil {
		return nil, err
	}
	for key, value := range p.options.headers {
		request.Header.Set(key, value)
	}
	// 设置 Request 中调用者传递的动态 Header
	for key, value := range req.Header {
		request.Header.Set(key, value)
	}
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", kiroCreds.AccessToken))
	request.Header.Set("amz-sdk-invocation-id", inv.ID)

	// 基于凭证信息生成稳定的动态指纹（User-Agent / x-amz-user-agent 等）
	accountKey := GetAccountKey(kiroCreds.ClientID, kiroCreds.RefreshToken)
	if accountKey == "" {
		accountKey = GenerateAccountKey(kiroCreds.AccessToken)
	}
	fp := GlobalFingerprintManager().GetFingerprint(accountKey)
	request.Header.Set("User-Agent", fp.BuildUserAgent())
	request.Header.Set("x-amz-user-agent", fp.BuildAmzUserAgent())

	// 3. 发送请求, 并检查状态码
	resp, err := p.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		return nil, &providers.HTTPError{
			ErrorType:       providers.ErrorTypeForbidden,
			ErrorCode:       resp.StatusCode,
			Message:         fmt.Sprintf("HTTP status code: %d", resp.StatusCode),
			RawStatusCode:   resp.StatusCode,
		}
	}

	// 4. 解码流式事件内容
	return p.handleStreamEvent(ctx, inv, resp.Body), nil
}

func (p *kiroProvider) handleStreamEvent(ctx context.Context, inv *providers.Invocation,
	respBody io.ReadCloser) queue.Consumer[*providers.Response] {
	chainQueue := queue.New[*providers.Response](defaultQueueSize)
	decoder := eventstream.NewDecoder()
	payloadBuf := make([]byte, defaultPayloadBufSize)
	var buf bytes.Buffer
	// 为每个GenerateContentStream调用创建新的ToolCall索引管理器
	toolCallIndexManager := parser.NewToolCallIndexManager()

	go func() {
		defer func() {
			chainQueue.Close()
			respBody.Close()
			log.InfoContextf(ctx, "\n ===== kiro stream event =====: %s", buf.String())
		}()
		
		// 收集用量统计信息，最后随 stop 事件一起发送
		var collectedUsage providers.Usage
		collectedUsage.PromptTokens = inv.Usage.PromptTokens
		var firstErr error
		// 跟踪上游返回的 stop_reason 和是否有工具调用
		var upstreamFinishReason string
		var hasToolCalls bool

		for {
			// 重置 payloadBuf 以复用底层数组
			payloadBuf = payloadBuf[0:0]
			e, err := decoder.Decode(respBody, payloadBuf)
			if err != nil {
				if err != io.EOF {
					firstErr = err
					log.Errorf("kiro stream decode error: %v", err)
				}
				break
			}

			msg := parser.StreamMessage(e)
			// 记录日志buf
			fmt.Fprintf(&buf, "\n[Event]: %s", msg.String())

			result, err := converter.ConvertResponse(ctx, &msg,
				parser.WithToolCallIndexManager(toolCallIndexManager))
			if err != nil || result == nil {
				continue
			}

			// 计算消耗token
			// if completionTokens, err := p.options.tokenConter.CountTokens(ctx, result.Message);
			// err == nil {
			// 	collectedUsage.PromptTokens += completionTokens
			// }
			collectedUsage.CompletionTokens += result.Usage.CompletionTokens
			// 检测到内联错误时，发送错误响应并终止流
			if result.Error != nil && result.Error.Message != "" {
				chainQueue.Push(ctx, result)
				firstErr = fmt.Errorf("kiro API error: %s", result.Error.Message)
				break
			}

			// 收集用量统计信息（metering/metadata 事件）
			if msg.IsMetricMessage() && result.Usage != nil {
				collectedUsage.Credit = result.Usage.Credit
			}
			if msg.IsMetadataMessage() && result.Usage != nil {
				// metadata 事件包含精确的 token 用量
				if result.Usage.CompletionTokens > 0 {
					collectedUsage.CompletionTokens = result.Usage.CompletionTokens
				}
				if result.Usage.PromptTokens > 0 {
					collectedUsage.PromptTokens = result.Usage.PromptTokens
				}
				if result.Usage.TotalTokens > 0 {
					collectedUsage.TotalTokens = result.Usage.TotalTokens
				}
				if result.Usage.PromptTokensDetails.CacheReadTokens > 0 {
					collectedUsage.PromptTokensDetails.CacheReadTokens = result.Usage.PromptTokensDetails.CacheReadTokens
					collectedUsage.PromptTokensDetails.CachedTokens = result.Usage.PromptTokensDetails.CachedTokens
				}
			}

			// 从每个 response 中收集 finish_reason（可能来自 assistantResponseEvent 或 messageStopEvent）
			for _, choice := range result.Choices {
				if choice.FinishReason != nil && *choice.FinishReason != "" {
					upstreamFinishReason = *choice.FinishReason
				}
			}

			// 跟踪是否有工具调用
			if result.IsToolCallResponse() {
				hasToolCalls = true
			}

			if msg.ShouldSendMessage() {
				chainQueue.Push(ctx, result)
			}
		}

		// 确定最终 finish_reason
		// 优先级：上游 stop_reason > 工具调用检测 > 默认 "stop"
		finishReason := upstreamFinishReason
		if finishReason == "" {
			if hasToolCalls {
				finishReason = "tool_calls"
			} else {
				finishReason = "stop"
			}
		}

		// 发送带有 usage 信息的 stop 响应
		if collectedUsage.TotalTokens == 0 {
			collectedUsage.TotalTokens = collectedUsage.PromptTokens + collectedUsage.CompletionTokens
		}
		finalResp := providers.NewResponse(ctx,
			providers.WithDone(true),
			providers.WithIsPartial(false),
			providers.WithUsage(&collectedUsage),
			providers.WithError(firstErr),
			providers.WithChoices(providers.Choice{
				FinishReason: &finishReason,
			}),
		)
		chainQueue.Push(ctx, finalResp)
	}()

	return chainQueue
}
