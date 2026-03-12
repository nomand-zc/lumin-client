package builder

import (
	"context"

	"github.com/nomand-zc/lumin-client/providers"
	"github.com/nomand-zc/lumin-client/providers/kiro/converter/builder/types"
)

// BuildContext 是各构建阶段共享的中间状态容器
// 每个 MessageBuilder 从中读取上一阶段的输出，并将本阶段的结果写入其中
type BuildContext struct {
	// 输入
	Ctx      context.Context
	Req      *providers.Request
	ModelId  string
	Metadata map[string]any

	// 阶段 1：预处理后的消息列表
	Messages []providers.Message

	// 阶段 2：提取出的 system prompt
	SystemPrompt string

	// 阶段 3：构建好的 kiro 工具列表
	KiroTools []types.Tool

	// 阶段 4：历史消息列表
	History []types.HistoryItem

	// 阶段 5：当前消息的各组成部分
	CurrentContent     string
	CurrentImages      []types.Image
	CurrentToolResults []types.ToolResult

	// PendingToolResults 是 HistoryBuilder 阶段未被消费的 tool_result
	// 这些 tool_result 来自最后消息之前的连续 tool 消息，需要传递给 CurrentMessageBuilder
	PendingToolResults []types.ToolResult

	// Done 为 true 时，流水线提前终止，ConvertRequest 返回 nil
	Done bool
}

// MessageBuilder 定义单个构建阶段的接口
// 每个实现负责一个职责单一的处理阶段，读写 BuildContext 中的字段
type MessageBuilder interface {
	Build(ctx *BuildContext) error
}
