package parser

import (
	"context"
	"encoding/json"

	"github.com/nomand-zc/lumin-client/providers"
)

// reasoningContentParser 处理 reasoningContentEvent 事件
// Kiro API 在 thinking 模式下返回此事件，包含模型的推理内容
// 格式: { "text": "...", "signature": "...", "redactedContent": "..." }
type reasoningContentParser struct{}

func init() {
	Register(&reasoningContentParser{})
}

func (p *reasoningContentParser) MessageType() string { return MessageTypeEvent }
func (p *reasoningContentParser) EventType() string   { return EventTypeReasoningContentEvent }

func (p *reasoningContentParser) Parse(ctx context.Context, msg *StreamMessage, opts ...OptionFunc) (*providers.Response, error) {
	var data struct {
		Text string `json:"text"`
		// Signature 和 RedactedContent 仅记录，不直接使用
		Signature       string `json:"signature,omitempty"`
		RedactedContent string `json:"redactedContent,omitempty"`
	}

	// 尝试从嵌套格式解析: { "reasoningContentEvent": { "text": "..." } }
	var nested map[string]json.RawMessage
	if err := json.Unmarshal(msg.Payload, &nested); err == nil {
		if inner, ok := nested["reasoningContentEvent"]; ok {
			if err := json.Unmarshal(inner, &data); err == nil && data.Text != "" {
				return buildReasoningResponse(ctx, data.Text), nil
			}
		}
	}

	// 直接格式解析: { "text": "..." }
	if err := json.Unmarshal(msg.Payload, &data); err != nil {
		return nil, nil
	}

	if data.Text == "" {
		return nil, nil
	}

	return buildReasoningResponse(ctx, data.Text), nil
}

func buildReasoningResponse(ctx context.Context, text string) *providers.Response {
	return providers.NewResponse(ctx,
		providers.WithChoices(providers.Choice{
			Index: 0,
			Delta: providers.Message{
				Role:             providers.RoleAssistant,
				ReasoningContent: text,
			},
		}),
	)
}
