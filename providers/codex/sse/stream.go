package sse

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/nomand-zc/lumin-client/providers"
	"github.com/nomand-zc/lumin-client/providers/codex/types"
	"github.com/nomand-zc/lumin-client/queue"
)

// StreamProcessor 将 Codex SSE 事件流转换为 providers.Response 队列
// 对齐 codex-rs/codex-api/src/sse/responses.rs 中 process_sse 和 process_responses_event 的实现
type StreamProcessor struct {
	model      string
	chainQueue queue.Queue[*providers.Response]
}

// NewStreamProcessor 创建 SSE 流处理器
func NewStreamProcessor(model string, chainQueue queue.Queue[*providers.Response]) *StreamProcessor {
	return &StreamProcessor{
		model:      model,
		chainQueue: chainQueue,
	}
}

// Process 解析 Codex SSE 流并将结果推送到队列
// 完全对齐 codex-rs/codex-api/src/sse/responses.rs 中的 process_sse 函数逻辑
func (sp *StreamProcessor) Process(ctx context.Context, body io.ReadCloser) {
	defer sp.chainQueue.Close()

	var collectedUsage providers.Usage
	var firstErr error
	var hasToolCalls bool
	var serverModel string
	var responseID string
	var reasoningOutputTokens int
	toolCallIndex := 0

	err := Parse(ctx, body, func(event Event) error {
		// 解析 SSE 事件为 ResponsesStreamEvent
		var sseEvent types.ResponsesStreamEvent
		if err := json.Unmarshal(event.Data, &sseEvent); err != nil {
			return nil // 忽略无法解析的事件
		}

		// 处理响应中嵌套的 headers 里的 openai-model
		if model := extractResponseModel(&sseEvent); model != "" && model != serverModel {
			serverModel = model
		}

		// 按事件类型分发处理
		// 对齐 codex-rs/codex-api/src/sse/responses.rs 中的 process_responses_event
		switch sseEvent.Type {
		case "response.created":
			// response.created 事件：对齐 codex 中直接返回 ResponseEvent::Created
			// 不需要推送到队列

		case "response.output_item.done":
			// response.output_item.done 事件：解析完成的输出项
			// 对齐 codex 中返回 ResponseEvent::OutputItemDone(item)
			if sseEvent.Item != nil {
				chunks := sp.processOutputItemDone(ctx, sseEvent.Item, &toolCallIndex, &hasToolCalls)
				for _, chunk := range chunks {
					if err := sp.chainQueue.Push(ctx, chunk); err != nil {
						return err
					}
				}
			}

		case "response.output_item.added":
			// response.output_item.added 事件：新输出项开始
			// 对齐 codex 中返回 ResponseEvent::OutputItemAdded(item)
			// 目前不需要特殊处理

		case "response.output_text.delta":
			// response.output_text.delta 事件：文本增量
			// 对齐 codex 中返回 ResponseEvent::OutputTextDelta(delta)
			if sseEvent.Delta != nil && *sseEvent.Delta != "" {
				chunk := providers.NewResponse(ctx,
					providers.WithModel(sp.model),
					providers.WithChoices(providers.Choice{
						Delta: providers.Message{
							Role:    providers.RoleAssistant,
							Content: *sseEvent.Delta,
						},
					}),
				)
				if err := sp.chainQueue.Push(ctx, chunk); err != nil {
					return err
				}
			}

		case "response.reasoning_summary_text.delta":
			// response.reasoning_summary_text.delta 事件：推理摘要增量
			// 对齐 codex 中返回 ResponseEvent::ReasoningSummaryDelta
			if sseEvent.Delta != nil && *sseEvent.Delta != "" {
				chunk := providers.NewResponse(ctx,
					providers.WithModel(sp.model),
					providers.WithChoices(providers.Choice{
						Delta: providers.Message{
							Role:             providers.RoleAssistant,
							ReasoningContent: *sseEvent.Delta,
						},
					}),
				)
				if err := sp.chainQueue.Push(ctx, chunk); err != nil {
					return err
				}
			}

		case "response.reasoning_text.delta":
			// response.reasoning_text.delta 事件：推理内容增量
			// 对齐 codex 中返回 ResponseEvent::ReasoningContentDelta
			if sseEvent.Delta != nil && *sseEvent.Delta != "" {
				chunk := providers.NewResponse(ctx,
					providers.WithModel(sp.model),
					providers.WithChoices(providers.Choice{
						Delta: providers.Message{
							Role:             providers.RoleAssistant,
							ReasoningContent: *sseEvent.Delta,
						},
					}),
				)
				if err := sp.chainQueue.Push(ctx, chunk); err != nil {
					return err
				}
			}

		case "response.reasoning_summary_part.added":
			// response.reasoning_summary_part.added 事件
			// 对齐 codex 中返回 ResponseEvent::ReasoningSummaryPartAdded
			// 目前不需要特殊处理

		case "response.completed":
			// response.completed 事件：响应完成，提取用量信息
			// 对齐 codex-rs/codex-api/src/sse/responses.rs 中 ResponseCompletedUsage -> TokenUsage 转换
			if sseEvent.Response != nil {
				var completed types.ResponseCompleted
				if err := json.Unmarshal(sseEvent.Response, &completed); err == nil {
					responseID = completed.ID
					if completed.Usage != nil {
						collectedUsage.PromptTokens = int(completed.Usage.InputTokens)
						collectedUsage.CompletionTokens = int(completed.Usage.OutputTokens)
						collectedUsage.TotalTokens = int(completed.Usage.TotalTokens)
						if completed.Usage.InputTokensDetails != nil {
							collectedUsage.PromptTokensDetails.CachedTokens = int(completed.Usage.InputTokensDetails.CachedTokens)
						}
						// 对齐 codex: 提取 reasoning_output_tokens
						if completed.Usage.OutputTokensDetails != nil {
							reasoningOutputTokens = int(completed.Usage.OutputTokensDetails.ReasoningTokens)
						}
					}
				}
			}

		case "response.failed":
			// response.failed 事件：响应失败
			// 对齐 codex 中各种错误分类逻辑
			if sseEvent.Response != nil {
				firstErr = parseResponseFailedError(sseEvent.Response)
			} else {
				firstErr = fmt.Errorf("response.failed event received")
			}
			return firstErr // 中断解析

		case "response.incomplete":
			// response.incomplete 事件：响应不完整
			// 对齐 codex 中的 ApiError::Stream 处理
			reason := "unknown"
			if sseEvent.Response != nil {
				var resp struct {
					IncompleteDetails *struct {
						Reason string `json:"reason"`
					} `json:"incomplete_details"`
				}
				if err := json.Unmarshal(sseEvent.Response, &resp); err == nil && resp.IncompleteDetails != nil {
					reason = resp.IncompleteDetails.Reason
				}
			}
			firstErr = fmt.Errorf("incomplete response returned, reason: %s", reason)
			return firstErr

		default:
			// 忽略未知事件类型
			// 对齐 codex 中的 trace!("unhandled responses event: {}", event.kind)
		}

		return nil
	})

	if err != nil && firstErr == nil {
		firstErr = fmt.Errorf("SSE stream error: %w", err)
	}

	// 确定最终 finish_reason
	finishReason := "stop"
	if hasToolCalls {
		finishReason = "tool_calls"
	}

	// 发送带有 usage 信息的最终 stop 响应
	if collectedUsage.TotalTokens == 0 {
		collectedUsage.TotalTokens = collectedUsage.PromptTokens + collectedUsage.CompletionTokens
	}
	// 对齐 codex: 设置 completion tokens details 中的 reasoning tokens
	if reasoningOutputTokens > 0 {
		collectedUsage.CompletionTokensDetails.ReasoningTokens = reasoningOutputTokens
	}

	finalModel := sp.model
	if serverModel != "" {
		finalModel = serverModel
	}

	finalResp := providers.NewResponse(ctx,
		providers.WithDone(true),
		providers.WithIsPartial(false),
		providers.WithUsage(&collectedUsage),
		providers.WithError(firstErr),
		providers.WithModel(finalModel),
		providers.WithChoices(providers.Choice{
			FinishReason: &finishReason,
		}),
	)
	if responseID != "" {
		finalResp.ID = responseID
	}

	if err := sp.chainQueue.Push(ctx, finalResp); err != nil {
		// 队列已关闭或 ctx 取消，finalResp 无法送达，与 kiro 侧保持一致
		_ = err
	}
}

// processOutputItemDone 处理 response.output_item.done 事件中的 item
// 对齐 codex-rs/codex-api/src/sse/responses.rs 中 response.output_item.done 分支
func (sp *StreamProcessor) processOutputItemDone(ctx context.Context, itemData json.RawMessage,
	toolCallIndex *int, hasToolCalls *bool) []*providers.Response {

	var item types.ResponseItemDone
	if err := json.Unmarshal(itemData, &item); err != nil {
		return nil
	}

	var chunks []*providers.Response

	switch item.Type {
	case "message":
		// 解析 message 类型的输出项
		// 对齐 codex protocol 中 ResponseItem::Message
		if item.Content != nil {
			var contentItems []types.OutputTextContent
			if err := json.Unmarshal(item.Content, &contentItems); err == nil {
				var text string
				for _, ci := range contentItems {
					if ci.Type == "output_text" {
						text += ci.Text
					}
				}
				if text != "" {
					chunks = append(chunks, providers.NewResponse(ctx,
						providers.WithModel(sp.model),
						providers.WithChoices(providers.Choice{
							Delta: providers.Message{
								Role:    providers.RoleAssistant,
								Content: text,
							},
						}),
					))
				}
			}
		}

	case "reasoning":
		// 解析 reasoning 类型的输出项
		// 对齐 codex protocol 中 ResponseItem::Reasoning
		if item.Summary != nil {
			var summaries []types.ReasoningSummaryText
			if err := json.Unmarshal(item.Summary, &summaries); err == nil {
				var reasoningText string
				for _, s := range summaries {
					if s.Type == "summary_text" {
						reasoningText += s.Text
					}
				}
				if reasoningText != "" {
					chunks = append(chunks, providers.NewResponse(ctx,
						providers.WithModel(sp.model),
						providers.WithChoices(providers.Choice{
							Delta: providers.Message{
								Role:             providers.RoleAssistant,
								ReasoningContent: reasoningText,
							},
						}),
					))
				}
			}
		}

	case "function_call":
		// 解析 function_call 类型的输出项
		// 对齐 codex protocol 中 ResponseItem::FunctionCall
		*hasToolCalls = true

		idx := *toolCallIndex
		*toolCallIndex++

		tc := providers.ToolCall{
			Type: "function",
			ID:   item.CallID,
			Function: providers.FunctionDefinitionParam{
				Name:      item.Name,
				Arguments: []byte(item.Arguments),
			},
			Index: &idx,
		}

		chunks = append(chunks, providers.NewResponse(ctx,
			providers.WithModel(sp.model),
			providers.WithChoices(providers.Choice{
				Delta: providers.Message{
					Role:      providers.RoleAssistant,
					ToolCalls: []providers.ToolCall{tc},
				},
			}),
		))

	case "web_search_call":
		// web_search_call 类型：codex 使用内置 web 搜索工具
		// 对齐 codex protocol 中 ResponseItem::WebSearchCall
		// 这里不需要特殊处理，当做信息事件

	case "image_generation_call":
		// image_generation_call 类型
		// 对齐 codex protocol 中 ResponseItem::ImageGenerationCall
	}

	return chunks
}

// parseResponseFailedError 解析 response.failed 事件中的错误
// 完全对齐 codex-rs/codex-api/src/sse/responses.rs 中 response.failed 的错误分类逻辑
func parseResponseFailedError(responseData json.RawMessage) error {
	var resp struct {
		Error *types.ResponseError `json:"error,omitempty"`
	}
	if err := json.Unmarshal(responseData, &resp); err != nil || resp.Error == nil {
		return fmt.Errorf("response.failed event received")
	}

	errObj := resp.Error

	switch errObj.Code {
	case "context_length_exceeded":
		// 对齐 codex 中 ApiError::ContextWindowExceeded
		return fmt.Errorf("context window exceeded: %s", errObj.Message)
	case "insufficient_quota":
		// 对齐 codex 中 ApiError::QuotaExceeded
		return fmt.Errorf("quota exceeded: %s", errObj.Message)
	case "usage_not_included":
		// 对齐 codex 中 ApiError::UsageNotIncluded
		return fmt.Errorf("usage not included: %s", errObj.Message)
	case "invalid_prompt":
		// 对齐 codex 中 ApiError::InvalidRequest
		msg := errObj.Message
		if msg == "" {
			msg = "Invalid request."
		}
		return fmt.Errorf("invalid prompt: %s", msg)
	case "server_is_overloaded", "slow_down":
		// 对齐 codex 中 ApiError::ServerOverloaded
		return fmt.Errorf("server overloaded: %s", errObj.Message)
	case "rate_limit_exceeded":
		// 对齐 codex 中 ApiError::Retryable，尝试从 message 中解析重试延迟
		if delay := tryParseRetryAfter(errObj.Message); delay > 0 {
			return fmt.Errorf("rate limit exceeded (retry after %v): %s", delay, errObj.Message)
		}
		return fmt.Errorf("rate limit exceeded: %s", errObj.Message)
	default:
		// 其他错误均视为可重试
		// 对齐 codex 中的 ApiError::Retryable 默认处理
		msg := errObj.Message
		if msg == "" {
			msg = "response.failed event received"
		}
		return fmt.Errorf("codex API error [%s]: %s", errObj.Code, msg)
	}
}

// tryParseRetryAfter 尝试从 rate_limit_exceeded 错误消息中解析重试延迟时间
// 对齐 codex-rs/codex-api/src/sse/responses.rs 中的 try_parse_retry_after 函数
func tryParseRetryAfter(message string) time.Duration {
	if message == "" {
		return 0
	}
	matches := _retryAfterRe.FindStringSubmatch(message)
	if len(matches) < 3 {
		return 0
	}
	value, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0
	}
	unit := strings.ToLower(matches[2])
	switch {
	case unit == "s" || strings.HasPrefix(unit, "second"):
		return time.Duration(value * float64(time.Second))
	case unit == "ms":
		return time.Duration(value * float64(time.Millisecond))
	}
	return 0
}

var _retryAfterRe = regexp.MustCompile(`(?i)try again in\s*(\d+(?:\.\d+)?)\s*(s|ms|seconds?)`)

// retryAfterRegex 返回用于解析 retry-after 的正则表达式（包级变量的 getter）
func retryAfterRegex() *regexp.Regexp {
	return _retryAfterRe
}

// extractResponseModel 从 SSE 事件中提取服务器返回的模型名称
// 对齐 codex-rs/codex-api/src/sse/responses.rs 中的 response_model() 方法
func extractResponseModel(event *types.ResponsesStreamEvent) string {
	// 优先从 response.headers 中获取
	if event.Response != nil {
		var resp struct {
			Headers map[string]any `json:"headers,omitempty"`
		}
		if err := json.Unmarshal(event.Response, &resp); err == nil && resp.Headers != nil {
			if model := findModelInHeaders(resp.Headers); model != "" {
				return model
			}
		}
	}

	// 其次从顶层 headers 中获取（websocket 元数据事件）
	if event.Headers != nil {
		var headers map[string]any
		if err := json.Unmarshal(event.Headers, &headers); err == nil {
			if model := findModelInHeaders(headers); model != "" {
				return model
			}
		}
	}

	return ""
}

// findModelInHeaders 在 headers map 中查找 openai-model 或 x-openai-model 头
// 对齐 codex: 使用忽略大小写的比较 (eq_ignore_ascii_case)
func findModelInHeaders(headers map[string]any) string {
	for key, val := range headers {
		lowKey := strings.ToLower(key)
		if lowKey == "openai-model" || lowKey == "x-openai-model" {
			if str, ok := val.(string); ok {
				return str
			}
			// 可能是数组形式
			if arr, ok := val.([]any); ok && len(arr) > 0 {
				if str, ok := arr[0].(string); ok {
					return str
				}
			}
		}
	}
	return ""
}
