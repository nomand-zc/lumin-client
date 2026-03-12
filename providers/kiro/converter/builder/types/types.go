package types

// ============================================================
// Kiro CodeWhisperer 请求结构体
// 对齐 buildCodewhispererRequest 函数的请求格式
// ============================================================

// Request 是发送给 Kiro CodeWhisperer API 的顶层请求结构
type Request struct {
	ConversationState ConversationState `json:"conversationState"`
	// ProfileArn 仅在 social 认证方式下存在
	ProfileArn string `json:"profileArn,omitempty"`
	// InferenceConfig 推理参数配置（maxTokens、temperature、topP）
	InferenceConfig *InferenceConfig `json:"inferenceConfig,omitempty"`
}

// InferenceConfig 推理参数配置
type InferenceConfig struct {
	MaxTokens   int     `json:"maxTokens,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
	TopP        float64 `json:"topP,omitempty"`
}

// ConversationState 描述当前对话状态
type ConversationState struct {
	AgentContinuationId string         `json:"agentContinuationId,omitempty"`
	AgentTaskType       string         `json:"agentTaskType,omitempty"`
	ChatTriggerType     string         `json:"chatTriggerType"`
	ConversationId      string         `json:"conversationId"`
	CurrentMessage      CurrentMessage `json:"currentMessage"`
	// History 仅在非空时存在（API 不接受空数组）
	History []HistoryItem `json:"history,omitempty"`
}

// CurrentMessage 当前轮次的消息，始终为 userInputMessage 类型
type CurrentMessage struct {
	UserInputMessage UserInputMessage `json:"userInputMessage"`
}

// UserInputMessage 用户输入消息
type UserInputMessage struct {
	Content string `json:"content"`
	ModelId string `json:"modelId"`
	Origin  string `json:"origin"`
	// Images 仅在有图片时存在
	Images []Image `json:"images,omitempty"`
	// UserInputMessageContext 仅在有 toolResults 或 tools 时存在
	UserInputMessageContext *UserInputMessageContext `json:"userInputMessageContext,omitempty"`
}

// UserInputMessageContext 用户输入消息的上下文，包含工具调用结果和工具列表
type UserInputMessageContext struct {
	// ToolResults 工具调用结果，仅在有结果时存在
	ToolResults []ToolResult `json:"toolResults,omitempty"`
	// Tools 工具列表，仅在有工具时存在
	Tools []Tool `json:"tools,omitempty"`
}

// HistoryItem 历史消息条目，userInputMessage 和 assistantResponseMessage 二选一
type HistoryItem struct {
	UserInputMessage         *UserInputMessage         `json:"userInputMessage,omitempty"`
	AssistantResponseMessage *AssistantResponseMessage `json:"assistantResponseMessage,omitempty"`
}

// AssistantResponseMessage 助手响应消息
type AssistantResponseMessage struct {
	Content string `json:"content"`
	// ToolUses 工具调用列表，仅在有工具调用时存在
	ToolUses []ToolUse `json:"toolUses,omitempty"`
}

// ToolUse 助手发起的工具调用
type ToolUse struct {
	Input     interface{} `json:"input"`
	Name      string      `json:"name"`
	ToolUseId string      `json:"toolUseId"`
}

// ToolResult 工具调用结果（用于 userInputMessageContext）
type ToolResult struct {
	Content   []ToolResultContent `json:"content"`
	Status    string              `json:"status"`
	ToolUseId string              `json:"toolUseId"`
}

// ToolResultContent 工具调用结果内容
type ToolResultContent struct {
	Text string `json:"text"`
}

// Tool 工具定义（用于 userInputMessageContext）
type Tool struct {
	ToolSpecification ToolSpecification `json:"toolSpecification"`
}

// ToolSpecification 工具规格定义
type ToolSpecification struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema InputSchema `json:"inputSchema"`
}

// InputSchema 工具输入 JSON Schema
type InputSchema struct {
	Json any `json:"json"`
}

// Image 图片数据（用于消息中的图片内容）
type Image struct {
	Format string      `json:"format"`
	Source ImageSource `json:"source"`
}

// ImageSource 图片来源（base64 编码的字节数据）
type ImageSource struct {
	Bytes string `json:"bytes"`
}
