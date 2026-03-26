package sse

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync/atomic"
	"time"

	"github.com/nomand-zc/lumin-client/providers"
	"github.com/nomand-zc/lumin-client/providers/geminicli/types"
	"github.com/nomand-zc/lumin-client/queue"
)

// StreamProcessor 将 Gemini SSE 事件流转换为 providers.Response 队列
type StreamProcessor struct {
	model       string
	chainQueue  queue.Queue[*providers.Response]
	toolCallSeq uint64 // 实例级别的 tool call ID 计数器
}

// NewStreamProcessor 创建 SSE 流处理器
func NewStreamProcessor(model string, chainQueue queue.Queue[*providers.Response]) *StreamProcessor {
	return &StreamProcessor{
		model:      model,
		chainQueue: chainQueue,
	}
}

// Process 解析 Gemini CLI SSE 流并将结果推送到队列
func (sp *StreamProcessor) Process(ctx context.Context, body io.ReadCloser) {
	defer sp.chainQueue.Close()

	var collectedUsage providers.Usage
	var firstErr error
	var hasToolCalls bool
	var upstreamFinishReason string
	var modelVersion string
	var responseID string
	toolCallIndex := 0

	err := Parse(ctx, body, func(event Event) error {
		// 解析 SSE 事件
		var sseResp types.GeminiCLIStreamResponse
		if err := json.Unmarshal(event.Data, &sseResp); err != nil {
			// 尝试解析为错误响应
			var errResp types.GeminiCLIErrorResponse
			if errErr := json.Unmarshal(event.Data, &errResp); errErr == nil && errResp.Error != nil {
				firstErr = fmt.Errorf("gemini API error [%d]: %s", errResp.Error.Code, errResp.Error.Message)
				return firstErr // 中断解析
			}
			return nil // 忽略无法解析的事件
		}

		if sseResp.Response == nil {
			return nil
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

			if candidate.FinishReason != "" {
				upstreamFinishReason = candidate.FinishReason
			}

			if candidate.Content != nil && len(candidate.Content.Parts) > 0 {
				for _, part := range candidate.Content.Parts {
					chunk := sp.convertPartToResponse(ctx, &part, responseID,
						&toolCallIndex, &hasToolCalls)
					if chunk != nil {
						sp.chainQueue.Push(ctx, chunk)
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
			if usage.CandidatesTokenCount > 0 {
				collectedUsage.CompletionTokens = usage.CandidatesTokenCount
			}
			if usage.TotalTokenCount > 0 {
				collectedUsage.TotalTokens = usage.TotalTokenCount
			}
		}

		return nil
	})

	if err != nil && firstErr == nil {
		firstErr = fmt.Errorf("SSE stream error: %w", err)
	}

	// 确定最终 finish_reason
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

	sp.chainQueue.Push(ctx, finalResp)
}

// convertPartToResponse 将单个 GeminiResponsePart 转换为 providers.Response
func (sp *StreamProcessor) convertPartToResponse(ctx context.Context, part *types.GeminiResponsePart,
	responseID string, toolCallIndex *int, hasToolCalls *bool) *providers.Response {

	// 处理 thinking 内容（thought=true 的文本）
	if part.Thought && part.Text != "" {
		return providers.NewResponse(ctx,
			providers.WithModel(sp.model),
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
			providers.WithModel(sp.model),
			providers.WithChoices(providers.Choice{
				Delta: providers.Message{
					Role:    providers.RoleAssistant,
					Content: part.Text,
				},
			}),
		)
	}

	// 处理函数调用
	if part.FunctionCall != nil {
		*hasToolCalls = true
		fc := part.FunctionCall

		var argsBytes []byte
		if fc.Args != nil {
			argsBytes, _ = json.Marshal(fc.Args)
		}

		// 使用实例级别计数器生成 tool call ID
		toolID := fmt.Sprintf("%s-%d-%d", fc.Name, time.Now().UnixNano(),
			atomic.AddUint64(&sp.toolCallSeq, 1))

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
			providers.WithModel(sp.model),
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
func mapFinishReason(geminiReason string, hasToolCalls bool) string {
	if hasToolCalls {
		return "tool_calls"
	}

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
