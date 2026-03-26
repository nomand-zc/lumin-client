package types

import "encoding/json"

// ResponsesAPIRequest 是发送到 Codex Responses API 的请求结构
// 完全对齐 codex-rs/codex-api/src/common.rs 中的 ResponsesApiRequest
type ResponsesAPIRequest struct {
	Model             string            `json:"model"`
	Instructions      string            `json:"instructions"`
	Input             []ResponseItem    `json:"input"`
	Tools             []json.RawMessage `json:"tools"`
	ToolChoice        string            `json:"tool_choice"`
	ParallelToolCalls bool              `json:"parallel_tool_calls"`
	Reasoning         *Reasoning        `json:"reasoning,omitempty"`
	Store             bool              `json:"store"`
	Stream            bool              `json:"stream"`
	Include           []string          `json:"include,omitempty"`
	ServiceTier       string            `json:"service_tier,omitempty"`
	PromptCacheKey    string            `json:"prompt_cache_key,omitempty"`
	Text              *TextControls     `json:"text,omitempty"`
}

// Reasoning 推理配置
type Reasoning struct {
	Effort  string `json:"effort,omitempty"`
	Summary string `json:"summary,omitempty"`
}

// TextControls 文本控制配置
type TextControls struct {
	Verbosity string      `json:"verbosity,omitempty"`
	Format    *TextFormat `json:"format,omitempty"`
}

// TextFormat 文本格式定义
type TextFormat struct {
	Type   string          `json:"type"`
	Strict bool            `json:"strict"`
	Schema json.RawMessage `json:"schema,omitempty"`
	Name   string          `json:"name,omitempty"`
}

// ResponseItem 是 Responses API 的输入/输出项
// 对齐 codex-rs/protocol/src/models.rs 中的 ResponseItem
type ResponseItem struct {
	Type string `json:"type"`

	// Message 类型字段
	ID      string        `json:"id,omitempty"`
	Role    string        `json:"role,omitempty"`
	Content []ContentItem `json:"content,omitempty"`

	// FunctionCall 类型字段
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
	CallID    string `json:"call_id,omitempty"`
	Status    string `json:"status,omitempty"`

	// FunctionCallOutput 类型字段
	Output string `json:"output,omitempty"`

	// Reasoning 类型字段
	Summary          []ReasoningSummaryItem `json:"summary,omitempty"`
	ReasoningContent []ReasoningContentItem `json:"reasoning_content,omitempty"`

	// 通用扩展字段
	ExtraFields map[string]any `json:"-"`
}

// ContentItem 表示消息内容项
type ContentItem struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
}

// ReasoningSummaryItem 推理摘要项
type ReasoningSummaryItem struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// ReasoningContentItem 推理内容项
type ReasoningContentItem struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// ToolFunction 工具函数定义，用于构建请求中的 tools 字段
type ToolFunction struct {
	Type     string            `json:"type"`
	Name     string            `json:"name,omitempty"`
	Function *ToolFunctionSpec `json:"function,omitempty"`
}

// ToolFunctionSpec 工具函数规格
type ToolFunctionSpec struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Strict      bool   `json:"strict,omitempty"`
	Parameters  any    `json:"parameters,omitempty"`
}
