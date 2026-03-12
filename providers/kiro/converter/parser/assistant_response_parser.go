package parser

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/nomand-zc/lumin-client/log"
	"github.com/nomand-zc/lumin-client/providers"
	"github.com/nomand-zc/lumin-client/utils"
)

// assistantResponseParser 处理 assistantResponseEvent 事件
type assistantResponseParser struct{}

func init() {
	Register(&assistantResponseParser{})
}

func (p *assistantResponseParser) MessageType() string { return MessageTypeEvent }
func (p *assistantResponseParser) EventType() string   { return EventTypeAssistantResponseEvent }

func (p *assistantResponseParser) Parse(ctx context.Context, msg *StreamMessage, opts ...OptionFunc) (*providers.Response, error) {
	payloadStr := utils.Bytes2Str(msg.Payload)

	// 检查是否是工具调用事件
	if isToolCallPayload(payloadStr) {
		return p.parseToolCall(ctx, msg, opts...)
	}

	// 尝试解析为 JSON
	var data struct {
		Content    string `json:"content"`
		StopReason string `json:"stop_reason"`
		StopRsn    string `json:"stopReason"`
	}
	if err := json.Unmarshal(msg.Payload, &data); err != nil {
		// 非 JSON 格式，作为纯文本内容处理
		text := strings.TrimSpace(payloadStr)
		if text == "" {
			return nil, nil
		}
		return providers.NewResponse(ctx,
			providers.WithChoices(providers.Choice{
				Index: 0,
				Delta: providers.Message{
					Role:    providers.RoleAssistant,
					Content: text,
				},
			}),
		), nil
	}

	// 提取 stop_reason（如果存在）
	stopReason := data.StopReason
	if stopReason == "" {
		stopReason = data.StopRsn
	}
	// 也尝试从嵌套格式提取
	if stopReason == "" {
		var nested map[string]json.RawMessage
		if json.Unmarshal(msg.Payload, &nested) == nil {
			if inner, ok := nested["assistantResponseEvent"]; ok {
				var innerData struct {
					StopReason string `json:"stop_reason"`
					StopRsn    string `json:"stopReason"`
				}
				if json.Unmarshal(inner, &innerData) == nil {
					stopReason = innerData.StopReason
					if stopReason == "" {
						stopReason = innerData.StopRsn
					}
				}
			}
		}
	}

	// 提取内容
	if data.Content == "" && stopReason == "" {
		return nil, nil
	}

	choice := providers.Choice{
		Index: 0,
		Delta: providers.Message{
			Role:    providers.RoleAssistant,
			Content: data.Content,
		},
	}
	if stopReason != "" {
		fr := MapKiroStopReasonToFinishReason(stopReason)
		choice.FinishReason = &fr
	}

	return providers.NewResponse(ctx,
		providers.WithChoices(choice),
	), nil
}

// parseToolCall 处理 assistantResponseEvent 中的工具调用
func (p *assistantResponseParser) parseToolCall(ctx context.Context, msg *StreamMessage, opts ...OptionFunc) (*providers.Response, error) {
	var evt struct {
		Name      string `json:"name"`
		ToolUseId string `json:"toolUseId"`
		Input     any    `json:"input"`
		Stop      bool   `json:"stop"`
	}
	if err := json.Unmarshal(msg.Payload, &evt); err != nil {
		log.Warnf("解析工具调用事件失败: %v", err)
		return nil, nil
	}

	// 解析选项参数
	parseOpt := &ParseOption{}
	for _, opt := range opts {
		opt(parseOpt)
	}

	// 使用ParseOption中的ToolCallIndexManager获取工具调用索引
	var index int
	if parseOpt.ToolCallIndexManager != nil {
		index = parseOpt.ToolCallIndexManager.GetToolCallIndex(evt.ToolUseId)
	}

	toolCall := providers.ToolCall{
		ID:    evt.ToolUseId,
		Type:  "function",
		Index: &index, // 设置正确的索引
		Function: providers.FunctionDefinitionParam{
			Name:      evt.Name,
			Arguments: convertInputToArgs(evt.Input),
		},
	}

	return providers.NewResponse(ctx,
		providers.WithObject(providers.ObjectChatCompletion),
		providers.WithIsPartial(!evt.Stop),
		providers.WithChoices(providers.Choice{
			Index: 0,
			Delta: providers.Message{
				Role:      providers.RoleAssistant,
				ToolCalls: []providers.ToolCall{toolCall},
			},
		}),
	), nil
}
