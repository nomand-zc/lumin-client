package builder

import (
	"github.com/google/uuid"
	"github.com/nomand-zc/lumin-client/providers/kiro/converter/builder/types"
)

const (
	// chatTriggerTypeManual 对话触发类型：手动触发
	chatTriggerTypeManual = "MANUAL"
	// originAIEditor 消息来源标识：AI 编辑器
	originAIEditor = "AI_EDITOR"
	// agentTaskTypeVibe Agent 任务类型
	agentTaskTypeVibe = "vibe"
	// kiroMaxOutputTokens Kiro API 最大输出 token 数（max_tokens=-1 时使用）
	kiroMaxOutputTokens = 32000
)

// Assemble 将 BuildContext 中的数据组装为最终的 *types.Request
func Assemble(ctx *BuildContext) *types.Request {
	req := &types.Request{}
	req.ConversationState.AgentTaskType = agentTaskTypeVibe
	req.ConversationState.ChatTriggerType = chatTriggerTypeManual
	req.ConversationState.ConversationId = uuid.NewString()

	// 组装 InferenceConfig（推理参数）
	if genCfg := ctx.Req.GenerationConfig; genCfg.MaxTokens != nil || genCfg.Temperature != nil || genCfg.TopP != nil {
		infCfg := &types.InferenceConfig{}
		if genCfg.MaxTokens != nil {
			maxTokens := *genCfg.MaxTokens
			if maxTokens == -1 {
				maxTokens = kiroMaxOutputTokens
			}
			infCfg.MaxTokens = maxTokens
		}
		if genCfg.Temperature != nil {
			infCfg.Temperature = *genCfg.Temperature
		}
		if genCfg.TopP != nil {
			infCfg.TopP = *genCfg.TopP
		}
		req.InferenceConfig = infCfg
	}

	// 构建 userInputMessage
	userInputMsg := types.UserInputMessage{
		Content: ctx.CurrentContent,
		ModelId: ctx.ModelId,
		Origin:  originAIEditor,
	}
	if len(ctx.CurrentImages) > 0 {
		userInputMsg.Images = ctx.CurrentImages
	}

	// 构建 userInputMessageContext
	userInputMsgCtx := &types.UserInputMessageContext{}
	hasCtx := false

	if len(ctx.CurrentToolResults) > 0 {
		// 过滤 currentToolResults 中的孤儿 tool_result
		// 截断历史后，某些 tool_result 对应的 assistant tool_use 可能已被移除
		filteredToolResults := filterOrphanedCurrentToolResults(ctx.CurrentToolResults, ctx.History)
		deduped := DeduplicateToolResults(filteredToolResults)
		if len(deduped) > 0 {
			userInputMsgCtx.ToolResults = deduped
			hasCtx = true
		}
	}
	if len(ctx.KiroTools) > 0 {
		userInputMsgCtx.Tools = ctx.KiroTools
		hasCtx = true
	}
	if hasCtx {
		userInputMsg.UserInputMessageContext = userInputMsgCtx
	}

	req.ConversationState.CurrentMessage.UserInputMessage = userInputMsg

	if len(ctx.History) > 0 {
		req.ConversationState.History = ctx.History
	}

	return req
}

// filterOrphanedCurrentToolResults 过滤 currentToolResults 中的孤儿 tool_result。
// 当历史截断导致产生 tool_use 的 assistant 消息被移除时，
// 当前消息中引用已被截断 assistant 的 tool_result 需要被清理。
func filterOrphanedCurrentToolResults(toolResults []types.ToolResult, history []types.HistoryItem) []types.ToolResult {
	if len(toolResults) == 0 {
		return toolResults
	}

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

	// 如果没有 history 中的 tool_use（可能是首次请求），保留所有 tool_result
	if len(validToolUseIDs) == 0 {
		return toolResults
	}

	filtered := make([]types.ToolResult, 0, len(toolResults))
	for _, tr := range toolResults {
		if validToolUseIDs[tr.ToolUseId] {
			filtered = append(filtered, tr)
		}
	}
	return filtered
}
