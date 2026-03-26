package builder

import (
	"fmt"
	"strings"

	"github.com/nomand-zc/lumin-client/providers"
	"github.com/nomand-zc/lumin-client/providers/kiro/converter/builder/types"
)

const (
	// keepImageThreshold 保留最近 N 条历史消息中的图片
	keepImageThreshold = 5
	// kiroMaxHistoryMessages 历史消息最大条数，防止超长历史导致 Kiro API 报错
	kiroMaxHistoryMessages = 50
	// defaultAssistantContent Kiro API 要求 assistant 消息内容非空时的最小兜底值
	// 使用 "." 而非有语义的句子，避免模型模仿回显
	defaultAssistantContent = "."
)

// HistoryBuilder 负责构建历史消息列表：
//   - 将除最后一条之外的所有消息转换为 types.HistoryItem
//   - 按 keepImageThreshold 策略决定是否保留历史消息中的图片
//   - 确保 history 以 user 消息开头（如有必要插入占位消息）
//   - 注意：systemPrompt 不在此处处理，统一由 Assembler 拼接到 currentMessage.content 中
//
// 结果写入 BuildContext.History
type HistoryBuilder struct{}

// Build 实现 MessageBuilder 接口
func (b *HistoryBuilder) Build(ctx *BuildContext) error {
	messages := ctx.Messages
	modelId := ctx.ModelId

	// 注意：systemPrompt 不再合并到 history 中。
	// 与 CLIProxyAPIPlus 对齐，systemPrompt（含 thinking 标签）统一在 Assembler 阶段
	// 拼接到 currentMessage.content 前面，使用 "--- SYSTEM PROMPT ---" 格式包装。

	history := []types.HistoryItem{}
	startIndex := 0

	// 构建历史消息（除最后一条之外的所有消息）
	// 使用 pendingToolResults 机制聚合连续的 tool 消息
	totalMessages := len(messages)
	var pendingToolResults []types.ToolResult

	for i := startIndex; i < totalMessages-1; i++ {
		msg := messages[i]
		// 计算距末尾的距离（从后往前数，最后一条消息距离为 0）
		distanceFromEnd := (totalMessages - 1) - i
		shouldKeepImages := distanceFromEnd <= keepImageThreshold

		switch msg.Role {
		case providers.RoleTool:
			// tool 消息先收集到 pendingToolResults 中，等后续 user 消息时一起合并
			pendingToolResults = append(pendingToolResults, types.ToolResult{
				ToolUseId: msg.ToolID,
				Status:    "success",
				Content:   []types.ToolResultContent{{Text: msg.Content}},
			})

		case providers.RoleUser:
			userInputMsg := BuildHistoryUserMessage(msg, modelId, shouldKeepImages)
			// 合并 pendingToolResults
			if len(pendingToolResults) > 0 {
				if userInputMsg.UserInputMessageContext == nil {
					userInputMsg.UserInputMessageContext = &types.UserInputMessageContext{}
				}
				userInputMsg.UserInputMessageContext.ToolResults = append(
					pendingToolResults, userInputMsg.UserInputMessageContext.ToolResults...)
				pendingToolResults = nil
			}
			history = append(history, types.HistoryItem{UserInputMessage: &userInputMsg})

		case providers.RoleAssistant:
			// 如果 assistant 消息前有未消费的 pendingToolResults，
			// 创建一条合成 user 消息来承载这些 tool_result
			if len(pendingToolResults) > 0 {
				syntheticUserMsg := types.UserInputMessage{
					Content: "Tool results provided.",
					ModelId: modelId,
					Origin:  originAIEditor,
					UserInputMessageContext: &types.UserInputMessageContext{
						ToolResults: pendingToolResults,
					},
				}
				history = append(history, types.HistoryItem{UserInputMessage: &syntheticUserMsg})
				pendingToolResults = nil
			}
			assistantMsg := BuildAssistantMessage(msg)
			history = append(history, types.HistoryItem{AssistantResponseMessage: &assistantMsg})
		}
	}

	// 如果末尾还有未消费的 pendingToolResults（最后一条消息之前的 tool 消息），
	// 传递给 CurrentMessageBuilder 阶段处理
	if len(pendingToolResults) > 0 {
		ctx.PendingToolResults = pendingToolResults
	}

	// Kiro API 要求 history 以 user 消息开头。
	// 某些客户端可能发送以 assistant 消息开头的对话，需要前置一条占位 user 消息。
	if len(history) > 0 && history[0].UserInputMessage == nil {
		placeholder := types.HistoryItem{
			UserInputMessage: &types.UserInputMessage{
				Content: ".",
				ModelId: modelId,
				Origin:  originAIEditor,
			},
		}
		history = append([]types.HistoryItem{placeholder}, history...)
	}

	// 截断超长历史消息
	history = truncateHistoryIfNeeded(history)
	// 过滤截断后的孤儿 ToolResult
	history = filterOrphanedToolResults(history)

	ctx.History = history
	return nil
}

// BuildHistoryUserMessage 构建历史中的 user 消息（支持图片保留策略）
func BuildHistoryUserMessage(msg providers.Message, modelId string, shouldKeepImages bool) types.UserInputMessage {
	userInputMsg := types.UserInputMessage{
		ModelId: modelId,
		Origin:  originAIEditor,
	}

	var sb strings.Builder
	var images []types.Image
	var toolResults []types.ToolResult
	imageCount := 0

	if msg.Role == providers.RoleTool {
		toolResults = append(toolResults, types.ToolResult{
			ToolUseId: msg.ToolID,
			Status:    "success",
			Content:   []types.ToolResultContent{{Text: msg.Content}},
		})
	} else if len(msg.ContentParts) > 0 {
		for _, part := range msg.ContentParts {
			switch part.Type {
			case providers.ContentTypeText:
				if part.Text != nil {
					sb.WriteString(*part.Text)
				}
			case providers.ContentTypeImage:
				if part.Image != nil {
					if shouldKeepImages {
						img := ConvertImage(part.Image)
						if img != nil {
							images = append(images, *img)
						}
					} else {
						imageCount++
					}
				}
			}
		}
	} else {
		sb.WriteString(msg.Content)
	}

	// 图片占位符
	if imageCount > 0 {
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString("[此消息包含 ")
		fmt.Fprintf(&sb, "%d", imageCount)
		sb.WriteString(" 张图片，已在历史记录中省略]")
	}

	textContent := sb.String()

	if len(toolResults) > 0 {
		userInputMsg.UserInputMessageContext = &types.UserInputMessageContext{
			ToolResults: DeduplicateToolResults(toolResults),
		}
		// Kiro API 要求 content 非空
		if strings.TrimSpace(textContent) == "" {
			userInputMsg.Content = "Tool results provided."
		} else {
			userInputMsg.Content = textContent
		}
	} else {
		// Kiro API 要求 content 非空
		if strings.TrimSpace(textContent) == "" {
			userInputMsg.Content = "Continue"
		} else {
			userInputMsg.Content = textContent
		}
	}

	if len(images) > 0 {
		userInputMsg.Images = images
	}

	return userInputMsg
}

// BuildAssistantMessage 将 providers.Message（assistant 角色）转换为 types.AssistantResponseMessage
// 支持 thinking 内容处理
func BuildAssistantMessage(msg providers.Message) types.AssistantResponseMessage {
	assistantMsg := types.AssistantResponseMessage{}

	var sb strings.Builder
	var toolUses []types.ToolUse
	var thinkingText string

	// 处理 ContentParts
	if len(msg.ContentParts) > 0 {
		for _, part := range msg.ContentParts {
			switch part.Type {
			case providers.ContentTypeText:
				if part.Text != nil {
					sb.WriteString(*part.Text)
				}
			}
		}
	} else {
		sb.WriteString(msg.Content)
	}

	// 处理 ReasoningContent（thinking）
	if msg.ReasoningContent != "" {
		thinkingText = msg.ReasoningContent
	}

	// 将 thinking 内容前置到 content
	var content string
	if thinkingText != "" {
		contentText := sb.String()
		var result strings.Builder
		result.WriteString("<thinking>")
		result.WriteString(thinkingText)
		result.WriteString("</thinking>")
		if contentText != "" {
			result.WriteString("\n\n")
			result.WriteString(contentText)
		}
		content = result.String()
	} else {
		content = sb.String()
	}

	// Kiro API 要求 assistant 消息内容非空
	if strings.TrimSpace(content) == "" {
		content = defaultAssistantContent
	}
	assistantMsg.Content = content

	// 处理工具调用
	if len(msg.ToolCalls) > 0 {
		for _, tc := range msg.ToolCalls {
			if tc.Type == "function" || tc.Type == "tool_use" {
				var input any
				if len(tc.Function.Arguments) > 0 {
					input = ParseJSONOrString(tc.Function.Arguments)
				} else {
					input = map[string]any{}
				}
				toolUses = append(toolUses, types.ToolUse{
					ToolUseId: tc.ID,
					Name:      tc.Function.Name,
					Input:     input,
				})
			}
		}
	}

	if len(toolUses) > 0 {
		assistantMsg.ToolUses = toolUses
	}

	return assistantMsg
}

// truncateHistoryIfNeeded 截断超长历史消息列表，保留最近的 kiroMaxHistoryMessages 条
func truncateHistoryIfNeeded(history []types.HistoryItem) []types.HistoryItem {
	if len(history) <= kiroMaxHistoryMessages {
		return history
	}
	return history[len(history)-kiroMaxHistoryMessages:]
}

// filterOrphanedToolResults 清理截断后的孤儿 ToolResult。
// 当历史截断导致产生 tool_use 的 assistant 消息被移除，
// 但后续的 user/tool_result 仍然存在时，需要移除这些无匹配的 tool_result。
func filterOrphanedToolResults(history []types.HistoryItem) []types.HistoryItem {
	// 收集所有有效的 toolUseId
	validToolUseIDs := make(map[string]bool)
	for _, h := range history {
		if h.AssistantResponseMessage == nil {
			continue
		}
		for _, tu := range h.AssistantResponseMessage.ToolUses {
			validToolUseIDs[tu.ToolUseId] = true
		}
	}

	// 过滤 history 中的 user 消息里的孤儿 tool_result
	for _, h := range history {
		if h.UserInputMessage == nil || h.UserInputMessage.UserInputMessageContext == nil {
			continue
		}
		ctx := h.UserInputMessage.UserInputMessageContext
		if len(ctx.ToolResults) == 0 {
			continue
		}

		filtered := make([]types.ToolResult, 0, len(ctx.ToolResults))
		for _, tr := range ctx.ToolResults {
			if validToolUseIDs[tr.ToolUseId] {
				filtered = append(filtered, tr)
			}
		}
		ctx.ToolResults = filtered
		if len(ctx.ToolResults) == 0 && len(ctx.Tools) == 0 {
			h.UserInputMessage.UserInputMessageContext = nil
		}
	}

	return history
}
