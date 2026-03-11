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
	"github.com/nomand-zc/lumin-client/credentials"
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
func (p *kiroProvider) GenerateContent(ctx context.Context, creds credentials.Credential, 
	req providers.Request) (*providers.Response, error) {
	reader, err := p.GenerateContentStream(ctx, creds, req)
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
func (p *kiroProvider) GenerateContentStream(ctx context.Context, creds credentials.Credential, 
	req providers.Request) (queue.Consumer[*providers.Response], error){
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
	kiroCreds := creds.(*kirocreds.Credential)
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
	for key, value := range p.options.headerBuilder() {
		request.Header.Set(key, value)
	}
	request.Header.Set("Accept", "text/event-stream")
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", kiroCreds.AccessToken))
	request.Header.Set("amz-sdk-invocation-id", inv.ID)
	// TODO: 需要根据creds生成稳定可靠的机器码，用于生成amz-sdk-invocation-id与x-amz-user-agent等信息

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

			// 如果是用量统计信息事件，收集起来而不直接发送
			if msg.IsMetricMessage() && result.Usage != nil {
				collectedUsage.Credit = result.Usage.Credit
			}

			if msg.ShouldSendMessage() {
chainQueue.Push(ctx, result)
			}
		}

		// 发送带有 usage 信息的 stop 响应
		collectedUsage.TotalTokens = collectedUsage.PromptTokens + collectedUsage.CompletionTokens
		finishReason := "stop"
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
