package builder

import (
	"strings"

	"github.com/nomand-zc/lumin-client/providers"
	"github.com/nomand-zc/lumin-client/providers/kiro/converter/builder/types"
)

// CurrentMessageBuilder 负责解析最后一条消息（currentMessage）：
//   - 若最后一条是 assistant 消息：将其加入 history，currentContent 设为 "Continue"
//   - 若最后一条是 user 消息（含 ContentParts）：分别提取文本和图片
//   - 若最后一条是 tool 消息：转换为 types.ToolResult
//   - currentContent 为空时的兜底逻辑
//
// 结果写入 BuildContext.CurrentContent、CurrentImages、CurrentToolResults
// 同时可能追加 BuildContext.History
type CurrentMessageBuilder struct{}

// Build 实现 MessageBuilder 接口
func (b *CurrentMessageBuilder) Build(ctx *BuildContext) error {
	messages := ctx.Messages
	totalMessages := len(messages)
	lastMsg := messages[totalMessages-1]

	var contentBuilder strings.Builder
	var currentToolResults []types.ToolResult
	var currentImages []types.Image

	if lastMsg.Role == providers.RoleAssistant {
		// 最后一条是 assistant 消息：将其加入 history，currentMessage 设为 "Continue"
		assistantMsg := BuildAssistantMessage(lastMsg)
		ctx.History = append(ctx.History, types.HistoryItem{AssistantResponseMessage: &assistantMsg})
		contentBuilder.WriteString("Continue")
	} else {
		// 最后一条是 user/tool 消息：确保 history 末尾是 assistantResponseMessage
		if len(ctx.History) > 0 {
			lastHistoryItem := ctx.History[len(ctx.History)-1]
			if lastHistoryItem.AssistantResponseMessage == nil {
				ctx.History = append(ctx.History, types.HistoryItem{
					AssistantResponseMessage: &types.AssistantResponseMessage{Content: "Continue"},
				})
			}
		}

		// 如果最后多条消息都是 tool 角色，收集它们作为 currentToolResults
		// 同时从 history 中找到这些连续 tool 消息之前的 tool 消息也一起收集
		if lastMsg.Role == providers.RoleTool {
			// 从最后一条往前扫描，收集连续的 tool 消息
			toolStartIdx := totalMessages - 1
			for toolStartIdx > 0 && messages[toolStartIdx-1].Role == providers.RoleTool {
				toolStartIdx--
			}
			// 收集所有尾部连续的 tool 消息作为 currentToolResults
			for i := toolStartIdx; i < totalMessages; i++ {
				msg := messages[i]
				currentToolResults = append(currentToolResults, types.ToolResult{
					ToolUseId: msg.ToolID,
					Status:    "success",
					Content:   []types.ToolResultContent{{Text: msg.Content}},
				})
			}
		}

		// 合并从 HistoryBuilder 传来的未消费的 PendingToolResults
		if len(ctx.PendingToolResults) > 0 {
			currentToolResults = append(ctx.PendingToolResults, currentToolResults...)
			ctx.PendingToolResults = nil // 清空，避免重复处理
		}

		// 解析最后一条 user 消息的内容
		if lastMsg.Role != providers.RoleTool {
			if len(lastMsg.ContentParts) > 0 {
				for _, part := range lastMsg.ContentParts {
					switch part.Type {
					case providers.ContentTypeText:
						if part.Text != nil {
							contentBuilder.WriteString(*part.Text)
						}
					case providers.ContentTypeImage:
						if part.Image != nil {
							img := ConvertImage(part.Image)
							if img != nil {
								currentImages = append(currentImages, *img)
							}
						}
					}
				}
			} else {
				contentBuilder.WriteString(lastMsg.Content)
			}
		}

		// content 兜底
		if contentBuilder.Len() == 0 {
			if len(currentToolResults) > 0 {
				contentBuilder.WriteString("Tool results provided.")
			} else {
				contentBuilder.WriteString("Continue")
			}
		}
	}

	ctx.CurrentContent = contentBuilder.String()
	ctx.CurrentImages = currentImages
	ctx.CurrentToolResults = currentToolResults
	return nil
}