package types

import "encoding/json"

// ResponsesStreamEvent 代表 SSE 流中的一个事件
// 对齐 codex-rs/codex-api/src/sse/responses.rs 中的 ResponsesStreamEvent
type ResponsesStreamEvent struct {
	// Type 事件类型，如 "response.created"、"response.output_item.done" 等
	Type string `json:"type"`
	// Response 响应体（用于 response.created、response.completed、response.failed 等事件）
	Response json.RawMessage `json:"response,omitempty"`
	// Item 输出项（用于 response.output_item.done、response.output_item.added 等事件）
	Item json.RawMessage `json:"item,omitempty"`
	// Delta 文本增量（用于 response.output_text.delta 等事件）
	Delta *string `json:"delta,omitempty"`
	// SummaryIndex 推理摘要索引
	SummaryIndex *int64 `json:"summary_index,omitempty"`
	// ContentIndex 推理内容索引
	ContentIndex *int64 `json:"content_index,omitempty"`
	// Headers 服务器返回的头信息（websocket元数据事件使用）
	Headers json.RawMessage `json:"headers,omitempty"`
}

// ResponseCompleted 对应 response.completed 事件中的 response 字段
// 对齐 codex-rs/codex-api/src/sse/responses.rs 中的 ResponseCompleted
type ResponseCompleted struct {
	ID    string                  `json:"id"`
	Usage *ResponseCompletedUsage `json:"usage,omitempty"`
}

// ResponseCompletedUsage 完成事件的用量信息
type ResponseCompletedUsage struct {
	InputTokens         int64                                 `json:"input_tokens"`
	InputTokensDetails  *ResponseCompletedInputTokensDetails  `json:"input_tokens_details,omitempty"`
	OutputTokens        int64                                 `json:"output_tokens"`
	OutputTokensDetails *ResponseCompletedOutputTokensDetails `json:"output_tokens_details,omitempty"`
	TotalTokens         int64                                 `json:"total_tokens"`
}

// ResponseCompletedInputTokensDetails 输入 token 详情
type ResponseCompletedInputTokensDetails struct {
	CachedTokens int64 `json:"cached_tokens"`
}

// ResponseCompletedOutputTokensDetails 输出 token 详情
type ResponseCompletedOutputTokensDetails struct {
	ReasoningTokens int64 `json:"reasoning_tokens"`
}

// ResponseError 响应失败事件中的错误信息
// 对齐 codex-rs/codex-api/src/sse/responses.rs 中的 Error
type ResponseError struct {
	Type     string `json:"type,omitempty"`
	Code     string `json:"code,omitempty"`
	Message  string `json:"message,omitempty"`
	PlanType string `json:"plan_type,omitempty"`
	ResetsAt *int64 `json:"resets_at,omitempty"`
}

// ResponseItemDone 对应 response.output_item.done 事件中的 item 字段
// 由于 item 类型多样，使用 json.RawMessage 延迟解析
type ResponseItemDone struct {
	Type string `json:"type"`
	// 以下字段根据 type 有选择地出现

	// message 类型
	ID      string          `json:"id,omitempty"`
	Role    string          `json:"role,omitempty"`
	Content json.RawMessage `json:"content,omitempty"`
	Phase   string          `json:"phase,omitempty"`

	// function_call 类型
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
	CallID    string `json:"call_id,omitempty"`
	Status    string `json:"status,omitempty"`

	// reasoning 类型
	Summary          json.RawMessage `json:"summary,omitempty"`
	ReasoningContent json.RawMessage `json:"reasoning_content,omitempty"`

	// web_search_call 类型
	Action json.RawMessage `json:"action,omitempty"`
}

// OutputTextContent 表示 output_text 类型的内容项
type OutputTextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ReasoningSummaryText 推理摘要文本项
type ReasoningSummaryText struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ReasoningText 推理内容文本项
type ReasoningText struct {
	Type string `json:"type"`
	Text string `json:"text"`
}
