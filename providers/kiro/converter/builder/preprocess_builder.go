package builder

import "github.com/nomand-zc/lumin-client/providers"

// PreprocessBuilder 负责消息预处理阶段：
//  1. 移除末尾内容为 "{" 的 assistant 消息
//  2. 合并相邻相同 role 的消息
//
// 处理结果写入 BuildContext.Messages；若结果为空则设置 Done = true
type PreprocessBuilder struct{}

// Build 实现 MessageBuilder 接口
func (b *PreprocessBuilder) Build(ctx *BuildContext) error {
	ctx.Messages = preprocessMessages(ctx.Req.Messages)
	if len(ctx.Messages) == 0 {
		ctx.Done = true
	}
	return nil
}

// preprocessMessages 预处理消息列表：
//  1. 移除末尾内容为 "{" 的 assistant 消息
//  2. 合并相邻相同 role 的消息
func preprocessMessages(messages []providers.Message) []providers.Message {
	if len(messages) == 0 {
		return messages
	}

	// Step 1: 检查并移除末尾内容为 "{" 的 assistant 消息
	result := make([]providers.Message, len(messages))
	copy(result, messages)

	last := result[len(result)-1]
	if last.Role == providers.RoleAssistant {
		content := last.Content
		if content == "" && len(last.ContentParts) > 0 {
			if last.ContentParts[0].Type == providers.ContentTypeText && last.ContentParts[0].Text != nil {
				content = *last.ContentParts[0].Text
			}
		}
		if content == "{" {
			result = result[:len(result)-1]
		}
	}

	if len(result) == 0 {
		return result
	}

	// Step 2: 合并相邻相同 role 的消息
	// 注意：tool 消息不参与合并，因为每条 tool 消息有独立的 ToolID，合并会丢失
	merged := []providers.Message{result[0]}
	for i := 1; i < len(result); i++ {
		cur := result[i]
		prev := &merged[len(merged)-1]

		// 不同 role 或 tool 角色消息不合并
		if cur.Role != prev.Role || cur.Role == providers.RoleTool {
			merged = append(merged, cur)
			continue
		}

		// 相同 role，合并内容
		prevIsArray := len(prev.ContentParts) > 0
		curIsArray := len(cur.ContentParts) > 0

		if prevIsArray && curIsArray {
			// 都是 ContentParts，直接追加
			prev.ContentParts = append(prev.ContentParts, cur.ContentParts...)
		} else if !prevIsArray && !curIsArray {
			// 都是 Content 字符串，用换行连接
			if prev.Content != "" && cur.Content != "" {
				prev.Content += "\n" + cur.Content
			} else if cur.Content != "" {
				prev.Content = cur.Content
			}
		} else if prevIsArray && !curIsArray {
			// prev 是数组，cur 是字符串，追加为 text part
			if cur.Content != "" {
				text := cur.Content
				prev.ContentParts = append(prev.ContentParts, providers.ContentPart{
					Type: providers.ContentTypeText,
					Text: &text,
				})
			}
		} else {
			// prev 是字符串，cur 是数组，转换 prev 为数组格式
			var newParts []providers.ContentPart
			if prev.Content != "" {
				text := prev.Content
				newParts = append(newParts, providers.ContentPart{
					Type: providers.ContentTypeText,
					Text: &text,
				})
			}
			newParts = append(newParts, cur.ContentParts...)
			prev.Content = ""
			prev.ContentParts = newParts
		}
	}

	return merged
}
