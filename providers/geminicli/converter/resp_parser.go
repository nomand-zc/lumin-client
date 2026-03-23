package converter

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync/atomic"
	"time"

	"github.com/nomand-zc/lumin-client/providers"
	"github.com/nomand-zc/lumin-client/queue"
)

// toolCallIDCounter 全局工具调用 ID 计数器
var toolCallIDCounter uint64

// ========== Gemini CLI API 响应结构体 ==========

// GeminiCLIStreamResponse SSE 流中每个 data: 行的顶层结构
type GeminiCLIStreamResponse struct {
	Response *GeminiCLIResponseBody `json:"response,omitempty"`
}

// GeminiCLIResponseBody 响应体
type GeminiCLIResponseBody struct {
	Candidates    []GeminiCandidate  `json:"candidates,omitempty"`
	UsageMetadata *GeminiUsage       `json:"usageMetadata,omitempty"`
	ModelVersion  string             `json:"modelVersion,omitempty"`
	ResponseID    string             `json:"responseId,omitempty"`
}

// GeminiCandidate 候选响应
type GeminiCandidate struct {
	Content      *GeminiCandidateContent `json:"content,omitempty"`
	FinishReason string                  `json:"finishReason,omitempty"`
}

// GeminiCandidateContent 候选内容
type GeminiCandidateContent struct {
	Parts []GeminiResponsePart `json:"parts,omitempty"`
	Role  string               `json:"role,omitempty"`
}

// GeminiResponsePart 响应中的一个部分
type GeminiResponsePart struct {
	Text             string                  `json:"text,omitempty"`
	Thought          bool                    `json:"thought,omitempty"`
	ThoughtSignature string                  `json:"thoughtSignature,omitempty"`
	FunctionCall     *GeminiResponseFuncCall `json:"functionCall,omitempty"`
}

// GeminiResponseFuncCall 响应中的函数调用
type GeminiResponseFuncCall struct {
	Name string         `json:"name"`
	Args map[string]any `json:"args,omitempty"`
}

// GeminiUsage 用量元数据
type GeminiUsage struct {
	PromptTokenCount     int `json:"promptTokenCount,omitempty"`
	CandidatesTokenCount int `json:"candidatesTokenCount,omitempty"`
	TotalTokenCount      int `json:"totalTokenCount,omitempty"`
	ThoughtsTokenCount   int `json:"thoughtsTokenCount,omitempty"`
}

// GeminiCLIErrorResponse Gemini CLI 错误响应
type GeminiCLIErrorResponse struct {
	Error *GeminiCLIError `json:"error,omitempty"`
}

// GeminiCLIError 错误详情
type GeminiCLIError struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
	Status  string `json:"status,omitempty"`
}

// ========== SSE 流解析 ==========

var dataPrefix = []byte("data: ")

// ParseSSEStream 解析 Gemini CLI SSE 流并将结果推送到队列
// 对齐 gemini-cli server.ts 中 requestStreamingPost 的 SSE 解析逻辑——支持多行 data: 拼接
func ParseSSEStream(ctx context.Context, body io.ReadCloser, model string,
	chainQueue queue.Queue[*providers.Response]) {
	defer func() {
		chainQueue.Close()
		body.Close()
	}()

	scanner := bufio.NewScanner(body)
	// 设置较大的缓冲区以处理大的 SSE 事件
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var collectedUsage providers.Usage
	var firstErr error
	var hasToolCalls bool
	var upstreamFinishReason string
	var modelVersion string
	var responseID string
	toolCallIndex := 0

	// 对齐 gemini-cli server.ts: 多行 data: 拼接缓冲区
	var bufferedLines []string

	for scanner.Scan() {
		line := scanner.Text()

		// 处理 data: 开头的行，累加到缓冲区
		if strings.HasPrefix(line, "data: ") {
			bufferedLines = append(bufferedLines, strings.TrimSpace(line[6:]))
			continue
		}

		// 空行表示一个 SSE 事件结束，对齐 gemini-cli server.ts 的 readline 逻辑
		if line == "" {
			if len(bufferedLines) == 0 {
				continue // 没有缓冲的数据要处理
			}

			// 拼接多行 data 并解析 JSON
			chunk := strings.Join(bufferedLines, "\n")
			bufferedLines = nil // 重置缓冲区

			jsonData := []byte(chunk)
			if len(bytes.TrimSpace(jsonData)) == 0 {
				continue
			}

			// 解析 SSE 事件
			var sseResp GeminiCLIStreamResponse
			if err := json.Unmarshal(jsonData, &sseResp); err != nil {
				// 尝试解析为错误响应
				var errResp GeminiCLIErrorResponse
				if errErr := json.Unmarshal(jsonData, &errResp); errErr == nil && errResp.Error != nil {
					firstErr = fmt.Errorf("gemini API error [%d]: %s", errResp.Error.Code, errResp.Error.Message)
					break
				}
			continue
			}

			if sseResp.Response == nil {
				continue
			}

			resp := sseResp.Response

			// 收集模型版本和响应 ID
			if resp.ModelVersion != "" {
				modelVersion = resp.ModelVersion
			}
			if resp.ResponseID != "" {
				responseID = resp.ResponseID
			}

			// 处理 candidates
			if len(resp.Candidates) > 0 {
				candidate := resp.Candidates[0]

				// 收集 finishReason
				if candidate.FinishReason != "" {
					upstreamFinishReason = candidate.FinishReason
				}

				if candidate.Content != nil && len(candidate.Content.Parts) > 0 {
					for _, part := range candidate.Content.Parts {
						chunk := convertPartToResponse(ctx, &part, model, responseID,
							&toolCallIndex, &hasToolCalls)
						if chunk != nil {
							chainQueue.Push(ctx, chunk)
						}
					}
				}
			}

			// 收集用量统计
			if resp.UsageMetadata != nil {
				usage := resp.UsageMetadata
				if usage.PromptTokenCount > 0 {
					collectedUsage.PromptTokens = usage.PromptTokenCount
				}
				// 对齐 gemini-cli: candidatesTokenCount 是生成的内容 token，
				// thoughtsTokenCount 是独立的思考 token 统计，不应简单相加
				if usage.CandidatesTokenCount > 0 {
					collectedUsage.CompletionTokens = usage.CandidatesTokenCount
				}
				if usage.TotalTokenCount > 0 {
					collectedUsage.TotalTokens = usage.TotalTokenCount
				}
			}
			continue
		}
		// 忽略其他行（如 event:、id:、注释等）
	}

	if err := scanner.Err(); err != nil {
		firstErr = fmt.Errorf("SSE stream scan error: %w", err)
	}

	// 确定最终 finish_reason，对齐 CLIProxyAPIPlus 中的逻辑
	finishReason := mapFinishReason(upstreamFinishReason, hasToolCalls)

	// 发送带有 usage 信息的最终 stop 响应
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
	if modelVersion != "" {
		finalResp.Model = modelVersion
	}

	chainQueue.Push(ctx, finalResp)
}

// convertPartToResponse 将单个 GeminiResponsePart 转换为 providers.Response
// 对齐 CLIProxyAPIPlus 中 ConvertGeminiCLIResponseToClaude 的 parts 处理逻辑
func convertPartToResponse(ctx context.Context, part *GeminiResponsePart,
	model, responseID string, toolCallIndex *int, hasToolCalls *bool) *providers.Response {

	// 处理 thinking 内容（thought=true 的文本）
	if part.Thought && part.Text != "" {
		return providers.NewResponse(ctx,
			providers.WithModel(model),
			providers.WithChoices(providers.Choice{
				Delta: providers.Message{
					Role:             providers.RoleAssistant,
					ReasoningContent: part.Text,
				},
			}),
		)
	}

	// 处理普通文本内容
	if part.Text != "" && !part.Thought {
		return providers.NewResponse(ctx,
			providers.WithModel(model),
			providers.WithChoices(providers.Choice{
				Delta: providers.Message{
					Role:    providers.RoleAssistant,
					Content: part.Text,
				},
			}),
		)
	}

	// 处理函数调用，对齐 CLIProxyAPIPlus 中 functionCall 的处理逻辑
	if part.FunctionCall != nil {
		*hasToolCalls = true
		fc := part.FunctionCall

		// 序列化 args
		var argsBytes []byte
		if fc.Args != nil {
			argsBytes, _ = json.Marshal(fc.Args)
		}

		// 生成 tool call ID，对齐 CLIProxyAPIPlus 中的 ID 生成逻辑
		toolID := fmt.Sprintf("%s-%d-%d", fc.Name, time.Now().UnixNano(),
			atomic.AddUint64(&toolCallIDCounter, 1))

		idx := *toolCallIndex
		*toolCallIndex++

		tc := providers.ToolCall{
			Type: "function",
			ID:   toolID,
			Function: providers.FunctionDefinitionParam{
				Name:      fc.Name,
				Arguments: argsBytes,
			},
			Index: &idx,
		}

		// 保存 thoughtSignature 到 ExtraFields，供后续多轮对话使用
		if part.ThoughtSignature != "" {
			tc.ExtraFields = map[string]any{
				"thoughtSignature": part.ThoughtSignature,
			}
		}

		return providers.NewResponse(ctx,
			providers.WithModel(model),
			providers.WithChoices(providers.Choice{
				Delta: providers.Message{
					Role:      providers.RoleAssistant,
					ToolCalls: []providers.ToolCall{tc},
				},
			}),
		)
	}

	return nil
}

// mapFinishReason 映射 Gemini finishReason 到统一格式
// 对齐 CLIProxyAPIPlus 中 stop_reason 映射逻辑
func mapFinishReason(geminiReason string, hasToolCalls bool) string {
	if hasToolCalls {
		return "tool_calls"
	}

	// 对齐 gemini-cli semantic.ts 中 toOTelFinishReason 的完整映射
	switch strings.ToUpper(geminiReason) {
	case "STOP", "FINISH_REASON_UNSPECIFIED", "UNKNOWN", "", "OTHER":
		return "stop"
	case "MAX_TOKENS":
		return "length"
	case "SAFETY", "RECITATION", "LANGUAGE", "BLOCKLIST", "PROHIBITED_CONTENT", "SPII", "IMAGE_SAFETY":
		return "content_filter"
	case "MALFORMED_FUNCTION_CALL", "UNEXPECTED_TOOL_CALL":
		return "error"
	default:
		return "stop"
	}
}

// ParseErrorResponse 解析 Gemini CLI 的非流式错误响应
func ParseErrorResponse(statusCode int, body []byte) error {
	var errResp GeminiCLIErrorResponse
	if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error != nil {
		msg := fmt.Sprintf("Gemini CLI API error [%d]: %s", errResp.Error.Code, errResp.Error.Message)

		switch statusCode {
		case 429:
			return &providers.HTTPError{
				ErrorType:     providers.ErrorTypeRateLimit,
				ErrorCode:     providers.ErrorCodeRateLimit,
				Message:       msg,
				RawStatusCode: statusCode,
				RawBody:       body,
			}
		case 401:
			return &providers.HTTPError{
				ErrorType:     providers.ErrorTypeUnauthorized,
				ErrorCode:     providers.ErrorCodeUnauthorized,
				Message:       msg,
				RawStatusCode: statusCode,
				RawBody:       body,
			}
		case 403:
			return &providers.HTTPError{
				ErrorType:     providers.ErrorTypeForbidden,
				ErrorCode:     providers.ErrorCodeForbidden,
				Message:       msg,
				RawStatusCode: statusCode,
				RawBody:       body,
			}
		case 400:
			return &providers.HTTPError{
				ErrorType:     providers.ErrorTypeBadRequest,
				ErrorCode:     providers.ErrorCodeBadRequest,
				Message:       msg,
				RawStatusCode: statusCode,
				RawBody:       body,
			}
		default:
			return &providers.HTTPError{
				ErrorType:     providers.ErrorTypeServerError,
				ErrorCode:     providers.ErrorCodeServerError,
				Message:       msg,
				RawStatusCode: statusCode,
				RawBody:       body,
			}
		}
	}

	return &providers.HTTPError{
		ErrorType:     providers.ErrorTypeServerError,
		ErrorCode:     providers.ErrorCodeServerError,
		Message:       fmt.Sprintf("Gemini CLI API error, status=%d", statusCode),
		RawStatusCode: statusCode,
		RawBody:       body,
	}
}
