package parser

import (
	"context"

	"github.com/nomand-zc/lumin-client/providers"
)

// noopParser 空操作解析器，用于已知但无需处理的事件类型
// 注册后可避免 resp_parser.go 中的 "未注册的事件解析器" 警告日志噪音
type noopParser struct {
	messageType string
	eventType   string
}

func init() {
	// invalidStateEvent: Kiro API 的状态提示事件，仅信息性，无需处理
	Register(&noopParser{messageType: MessageTypeEvent, eventType: EventTypeInvalidStateEvent})
	// followupPromptEvent: Kiro API 的跟进提示建议，属于 UI 层建议，无需处理
	Register(&noopParser{messageType: MessageTypeEvent, eventType: EventTypeFollowupPromptEvent})
}

func (p *noopParser) MessageType() string { return p.messageType }
func (p *noopParser) EventType() string   { return p.eventType }

func (p *noopParser) Parse(ctx context.Context, msg *StreamMessage, opts ...OptionFunc) (*providers.Response, error) {
	// 空操作，返回 nil 表示该事件不产生输出
	return nil, nil
}
