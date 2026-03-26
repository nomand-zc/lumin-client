package parser

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nomand-zc/lumin-client/providers"
	"github.com/nomand-zc/lumin-client/utils"
)

// completionParser 处理 completion 事件（完整响应）
type completionParser struct{}

func init() {
	Register(&completionParser{})
}

func (p *completionParser) MessageType() string { return MessageTypeEvent }
func (p *completionParser) EventType() string   { return EventTypeCompletion }

func (p *completionParser) Parse(ctx context.Context, msg *StreamMessage, opts ...OptionFunc) (*providers.Response, error) {
	var data struct {
		Content      string `json:"content"`
		FinishReason string `json:"finish_reason"`
		ToolCalls    []struct {
			ID       string `json:"id"`
			Type     string `json:"type"`
			Function struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			} `json:"function"`
		} `json:"tool_calls"`
	}
	if err := json.Unmarshal(msg.Payload, &data); err != nil {
		return nil, fmt.Errorf("解析 completion 事件载荷失败: %w", err)
	}

	resp := providers.NewResponse(ctx,
		providers.WithObject(providers.ObjectChatCompletion),
		providers.WithIsPartial(false),
		providers.WithDone(true),
		providers.WithChoices(providers.Choice{
			Index: 0,
			Delta: providers.Message{
				Role:    providers.RoleAssistant,
				Content: data.Content,
			},
		}),
	)

	// 设置完成原因
	if data.FinishReason != "" {
		resp.Choices[0].FinishReason = &data.FinishReason
	}

	// 处理工具调用
	if len(data.ToolCalls) > 0 {
		toolCalls := make([]providers.ToolCall, 0, len(data.ToolCalls))
		for _, tc := range data.ToolCalls {
			toolCalls = append(toolCalls, providers.ToolCall{
				ID:   tc.ID,
				Type: tc.Type,
				Function: providers.FunctionDefinitionParam{
					Name:      tc.Function.Name,
					Arguments: utils.Str2Bytes(tc.Function.Arguments),
				},
			})
		}
		resp.Choices[0].Delta.ToolCalls = toolCalls
		resp.Done = false
	}

	return resp, nil
}
