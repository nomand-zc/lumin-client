package converter

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nomand-zc/lumin-client/log"
	"github.com/nomand-zc/lumin-client/providers"
	"github.com/nomand-zc/lumin-client/providers/kiro/converter/parser"
	"github.com/nomand-zc/lumin-client/utils"
)

// ConvertResponse 将 Kiro CodeWhisperer 响应转换为通用响应格式
// 通过 parser 注册器根据 messageType 和 eventType 获取对应的解析器来处理
func ConvertResponse(ctx context.Context, resp *parser.StreamMessage, opts ...parser.OptionFunc) (
	*providers.Response, error) {
	if resp == nil {
		return nil, nil
	}

	messageType := resp.MessageType()
	eventType := resp.EventType()

	// 全局错误检测：Kiro API 可能以 HTTP 200 返回内联错误
	// 在分发到具体 parser 之前，先检查 payload 中的错误标记
	if errResp := detectInlineError(ctx, resp.Payload, messageType, eventType); errResp != nil {
		return errResp, nil
	}

	// 优先尝试 messageType+eventType 组合查找（适用于 event 类型消息）
	p := parser.Get(messageType, eventType)
	if p == nil {
		// 对于未注册的 event 子类型，记录日志并忽略
		log.Warnf("未注册的事件解析器: messageType=%s, eventType=%s, payload: %s",
			messageType, eventType, utils.Bytes2Str(resp.Payload))
		return nil, nil
	}

	return p.Parse(ctx, resp, opts...)
}

// detectInlineError 检测 Kiro API 以 HTTP 200 内联返回的错误
// 支持两种格式：
//  1. AWS 格式：payload 包含 "_type" 字段（如 "com.amazon.aws.codewhisperer#ValidationException"）
//  2. 通用格式：payload 包含 "type" 字段值为 "error"/"exception"
//  3. 事件类型为 error/exception/internalServerException
func detectInlineError(ctx context.Context, payload []byte, messageType, eventType string) *providers.Response {
	if len(payload) == 0 {
		return nil
	}

	// 对 event 类型消息，检查 eventType 是否为错误类型
	isErrorEventType := eventType == parser.EventTypeError ||
		eventType == parser.EventTypeException ||
		eventType == parser.EventTypeInternalServerException

	// 快速判断：如果不是错误 messageType 也不是错误 eventType，尝试 payload 探测
	// 避免对每条正常消息都做 JSON unmarshal
	if messageType != parser.MessageTypeError && messageType != parser.MessageTypeException && !isErrorEventType {
		// 快速字节检测：payload 是否可能包含 _type 或 "type":"error"
		if !containsErrorMarker(payload) {
			return nil
		}
	}

	var event map[string]any
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil
	}

	// 检查 AWS 格式：{"_type": "com.amazon.aws.codewhisperer#ValidationException", "message": "..."}
	if errType, ok := event["_type"].(string); ok && errType != "" {
		errMsg, _ := utils.GetMapValue[string, string](event, "message")
		log.Errorf("检测到 AWS 内联错误: _type=%s, message=%s", errType, errMsg)
		return providers.NewResponse(ctx,
			providers.WithResponseError(&providers.ResponseError{
				Message: errMsg,
				Type:    "error",
				Code:    utils.ToPtr(errType),
			}),
		)
	}

	// 检查通用格式：{"type": "error", "message": "..."} 或 {"type": "exception", ...}
	if typeVal, ok := event["type"].(string); ok && (typeVal == "error" || typeVal == "exception") {
		errMsg, _ := utils.GetMapValue[string, string](event, "message")
		// 尝试从嵌套 error 对象中获取 message
		if errMsg == "" {
			if errObj, ok := event["error"].(map[string]any); ok {
				errMsg, _ = utils.GetMapValue[string, string](errObj, "message")
			}
		}
		log.Errorf("检测到通用内联错误: type=%s, message=%s", typeVal, errMsg)
		return providers.NewResponse(ctx,
			providers.WithResponseError(&providers.ResponseError{
				Message: errMsg,
				Type:    typeVal,
			}),
		)
	}

	// 处理 error/exception/internalServerException 事件类型
	if isErrorEventType {
		errMsg, _ := utils.GetMapValue[string, string](event, "message")
		// 尝试从嵌套的事件对象中获取
		if errMsg == "" {
			if nested, ok := event[eventType].(map[string]any); ok {
				errMsg, _ = utils.GetMapValue[string, string](nested, "message")
			}
		}
		if errMsg == "" {
			errMsg = fmt.Sprintf("kiro API error event: %s", eventType)
		}
		log.Errorf("检测到错误事件: eventType=%s, message=%s", eventType, errMsg)
		return providers.NewResponse(ctx,
			providers.WithResponseError(&providers.ResponseError{
				Message: errMsg,
				Type:    eventType,
			}),
		)
	}

	return nil
}

// containsErrorMarker 快速字节检测 payload 是否可能包含错误标记
// 避免对每条正常消息都做完整的 JSON unmarshal·
func containsErrorMarker(payload []byte) bool {
	s := utils.Bytes2Str(payload)
	return len(s) > 0 && (
	// AWS 格式
	strings.Contains(s, `"_type"`) ||
		// 通用格式
		strings.Contains(s, `"type":"error"`) ||
		strings.Contains(s, `"type":"exception"`) ||
		strings.Contains(s, `"type": "error"`) ||
		strings.Contains(s, `"type": "exception"`))
}
