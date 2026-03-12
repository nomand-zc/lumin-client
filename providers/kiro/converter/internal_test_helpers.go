package converter

// 本文件为 converter 包内部测试提供辅助函数包装
// 将 builder 包中的内部函数桥接到 converter 包，供白盒测试（package converter）直接调用

import (
	"github.com/nomand-zc/lumin-client/providers"
	"github.com/nomand-zc/lumin-client/providers/kiro/converter/builder"
)

const (
	// maxDescriptionLength 工具描述最大长度（与 builder 包保持一致）
	maxDescriptionLength = 9216
)

// preprocessMessages 包装 builder.PreprocessBuilder 的内部逻辑，供测试使用
func preprocessMessages(messages []providers.Message) []providers.Message {
	bCtx := &builder.BuildContext{
		Req: &providers.Request{Messages: messages},
	}
	b := &builder.PreprocessBuilder{}
	_ = b.Build(bCtx)
	return bCtx.Messages
}

// convertImage 包装 builder.ConvertImage，供测试使用
func convertImage(img *providers.Image) *Image {
	return builder.ConvertImage(img)
}

// buildKiroTools 包装 builder.ToolsBuilder 的内部逻辑，供测试使用
func buildKiroTools(tools []providers.Tool) []Tool {
	bCtx := &builder.BuildContext{
		Req: &providers.Request{Tools: tools},
	}
	b := &builder.ToolsBuilder{}
	_ = b.Build(bCtx)
	return bCtx.KiroTools
}

// buildAssistantMessage 包装 builder.BuildAssistantMessage，供测试使用
func buildAssistantMessage(msg providers.Message) AssistantResponseMessage {
	return builder.BuildAssistantMessage(msg)
}

// deduplicateToolResults 包装 builder.DeduplicateToolResults，供测试使用
func deduplicateToolResults(toolResults []ToolResult) []ToolResult {
	return builder.DeduplicateToolResults(toolResults)
}
