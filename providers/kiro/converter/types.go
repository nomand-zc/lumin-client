package converter

import "github.com/nomand-zc/lumin-client/providers/kiro/converter/builder/types"

// 以下类型别名将 types 子包的类型重新导出到 converter 包
// 保持外部调用方的兼容性，无需修改任何引用 converter.Xxx 的代码

type KiroRequest = types.Request
type ConversationState = types.ConversationState
type CurrentMessage = types.CurrentMessage
type UserInputMessage = types.UserInputMessage
type UserInputMessageContext = types.UserInputMessageContext
type HistoryItem = types.HistoryItem
type AssistantResponseMessage = types.AssistantResponseMessage
type ToolUse = types.ToolUse
type ToolResult = types.ToolResult
type ToolResultContent = types.ToolResultContent
type Tool = types.Tool
type ToolSpecification = types.ToolSpecification
type InputSchema = types.InputSchema
type Image = types.Image
type ImageSource = types.ImageSource
type InferenceConfig = types.InferenceConfig
