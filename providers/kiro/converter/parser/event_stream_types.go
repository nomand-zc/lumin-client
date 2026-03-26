package parser

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream"
	"github.com/nomand-zc/lumin-client/utils"
)

// 消息类型常量
const (
	MessageTypeEvent     = "event"
	MessageTypeError     = "error"
	MessageTypeException = "exception"
)

// 事件类型常量
const (
	// 代码补全
	EventTypeCompletion      = "completion"
	EventTypeCompletionChunk = "completion_chunk"

	// 工具调用相关
	EventTypeToolCallRequest    = "tool_call_request"
	EventTypeToolCallResult     = "tool_call_result"
	EventTypeToolCallError      = "tool_call_error"
	EventTypeToolExecutionStart = "tool_execution_start"
	EventTypeToolExecutionEnd   = "tool_execution_end"

	// 会话管理
	EventTypeSessionStart = "session_start"
	EventTypeSessionEnd   = "session_end"

	// 统计信息
	EventTypeMeteringEvent     = "meteringEvent"
	EventTypeContextUsageEvent = "contextUsageEvent"

	// 兼容旧格式
	EventTypeAssistantResponseEvent = "assistantResponseEvent"
	EventTypeToolUseEvent           = "toolUseEvent"

	// Kiro 原生事件类型
	EventTypeReasoningContentEvent = "reasoningContentEvent"
	EventTypeMessageMetadataEvent  = "messageMetadataEvent"
	EventTypeMetadataEvent         = "metadataEvent"
	EventTypeMessageStopEvent      = "messageStopEvent"
	EventTypeInvalidStateEvent     = "invalidStateEvent"
	EventTypeFollowupPromptEvent   = "followupPromptEvent"
	EventTypeSupplementaryWebLinks = "supplementaryWebLinksEvent"

	// 错误事件类型（Kiro API 可能以 event 类型返回错误）
	EventTypeError                   = "error"
	EventTypeException               = "exception"
	EventTypeInternalServerException = "internalServerException"

	// 用量事件类型
	EventTypeUsageEvent = "usageEvent"
	EventTypeUsage      = "usage"
)

// StreamMessage 流式消息
type StreamMessage eventstream.Message

func (m StreamMessage) MessageType() string {
	if v := m.Headers.Get(":message-type"); v != nil {
		if sv, ok := v.(eventstream.StringValue); ok {
			return string(sv)
		}
	}
	return MessageTypeEvent // 默认为事件类型
}

// EventType 事件类型
func (m StreamMessage) EventType() string {
	if v := m.Headers.Get(":event-type"); v != nil {
		if sv, ok := v.(eventstream.StringValue); ok {
			return string(sv)
		}
	}
	return ""
}

// GetContentType 从头部提取内容类型
func (m StreamMessage) GetContentType() string {
	if v := m.Headers.Get(":content-type"); v != nil {
		if sv, ok := v.(eventstream.StringValue); ok {
			return string(sv)
		}
	}
	return "application/json" // 默认为JSON
}

// IsMetricMessage 是否为统计信息消息
func (m StreamMessage) IsMetricMessage() bool {
	return m.MessageType() == MessageTypeEvent && m.EventType() == EventTypeMeteringEvent
}

// IsContextUsageMessage 是否为上下文使用量消息
func (m StreamMessage) IsContextUsageMessage() bool {
	return m.MessageType() == MessageTypeEvent && m.EventType() == EventTypeContextUsageEvent
}

// IsMetadataMessage 是否为元数据消息（包含 token usage 等信息）
func (m StreamMessage) IsMetadataMessage() bool {
	et := m.EventType()
	return m.MessageType() == MessageTypeEvent &&
		(et == EventTypeMessageMetadataEvent || et == EventTypeMetadataEvent ||
			et == EventTypeSupplementaryWebLinks || et == EventTypeUsageEvent || et == EventTypeUsage)
}

// ShouldSendMessage 是否应该发送消息
func (m StreamMessage) ShouldSendMessage() bool {
	return !m.IsMetricMessage() && !m.IsContextUsageMessage() && !m.IsMetadataMessage() && len(m.Payload) > 0
}

// String 消息字符串
func (m StreamMessage) String() string {
	return fmt.Sprintf("Headers: %+v, Payload: %s", m.Headers, utils.Bytes2Str(m.Payload))
}
