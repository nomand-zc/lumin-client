package parser

import (
	"context"
	"encoding/json"

	"github.com/nomand-zc/lumin-client/providers"
)

// messageStopParser 处理 messageStopEvent 事件
// Kiro API 在消息结束时返回此事件，包含 stop_reason
type messageStopParser struct{}

func init() {
	Register(&messageStopParser{})
}

func (p *messageStopParser) MessageType() string { return MessageTypeEvent }
func (p *messageStopParser) EventType() string   { return EventTypeMessageStopEvent }

func (p *messageStopParser) Parse(ctx context.Context, msg *StreamMessage, opts ...OptionFunc) (*providers.Response, error) {
	stopReason := extractStopReason(msg.Payload)
	if stopReason == "" {
		return nil, nil
	}

	finishReason := MapKiroStopReasonToFinishReason(stopReason)
	return providers.NewResponse(ctx,
		providers.WithChoices(providers.Choice{
			Index:        0,
			FinishReason: &finishReason,
		}),
	), nil
}

// extractStopReason 从 payload 中提取 stop_reason，支持多种格式
func extractStopReason(payload []byte) string {
	var data map[string]interface{}
	if err := json.Unmarshal(payload, &data); err != nil {
		return ""
	}

	// 检查顶层 stop_reason / stopReason
	if sr, ok := data["stop_reason"].(string); ok && sr != "" {
		return sr
	}
	if sr, ok := data["stopReason"].(string); ok && sr != "" {
		return sr
	}

	// 检查嵌套格式 { "messageStopEvent": { "stopReason": "..." } }
	if inner, ok := data["messageStopEvent"].(map[string]interface{}); ok {
		if sr, ok := inner["stop_reason"].(string); ok && sr != "" {
			return sr
		}
		if sr, ok := inner["stopReason"].(string); ok && sr != "" {
			return sr
		}
	}

	return ""
}

// MapKiroStopReasonToFinishReason 将 Kiro/Claude stop_reason 转换为 OpenAI 风格 finish_reason
func MapKiroStopReasonToFinishReason(stopReason string) string {
	switch stopReason {
	case "end_turn":
		return "stop"
	case "stop_sequence":
		return "stop"
	case "tool_use":
		return "tool_calls"
	case "max_tokens":
		return "length"
	case "content_filtered":
		return "content_filter"
	default:
		return stopReason
	}
}
