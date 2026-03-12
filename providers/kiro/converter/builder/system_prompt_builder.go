package builder

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nomand-zc/lumin-client/providers"
)

// SystemPromptBuilder 负责从消息列表中提取 system prompt：
//   - 将所有 system 消息的内容用 "\n\n" 合并，写入 BuildContext.SystemPrompt
//   - 注入 thinking 模式标记（如果 GenerationConfig 中启用）
//   - 注入 tool_choice 和 response_format 提示（如果 Metadata 中存在）
//   - 将非 system 消息写回 BuildContext.Messages
//   - 若过滤后消息列表为空，则设置 Done = true
type SystemPromptBuilder struct{}

// Build 实现 MessageBuilder 接口
func (b *SystemPromptBuilder) Build(ctx *BuildContext) error {
	var sb strings.Builder
	var nonSystemMessages []providers.Message

	for _, msg := range ctx.Messages {
		if msg.Role == providers.RoleSystem {
			if msg.Content != "" {
				if sb.Len() > 0 {
					sb.WriteString("\n\n")
				}
				sb.WriteString(msg.Content)
			}
		} else {
			nonSystemMessages = append(nonSystemMessages, msg)
		}
	}

	systemPrompt := sb.String()

	// 注入 thinking 模式标记
	systemPrompt = injectThinkingMode(ctx.Req, systemPrompt)

	// 注入 tool_choice 提示
	if ctx.Metadata != nil {
		if hint := extractToolChoiceHint(ctx.Metadata); hint != "" {
			if systemPrompt != "" {
				systemPrompt += "\n"
			}
			systemPrompt += hint
		}
	}

	// 注入 response_format 提示
	if ctx.Metadata != nil {
		if hint := extractResponseFormatHint(ctx.Metadata); hint != "" {
			if systemPrompt != "" {
				systemPrompt += "\n"
			}
			systemPrompt += hint
		}
	}

	ctx.SystemPrompt = systemPrompt
	ctx.Messages = nonSystemMessages

	if len(ctx.Messages) == 0 {
		ctx.Done = true
	}
	return nil
}

// injectThinkingMode 根据 GenerationConfig 注入 thinking 模式标记到 system prompt
// Kiro API 通过 <thinking_mode> 和 <max_thinking_length> 标签来启用 thinking 模式
func injectThinkingMode(req *providers.Request, systemPrompt string) string {
	if req == nil {
		return systemPrompt
	}

	cfg := req.GenerationConfig
	thinkingEnabled := false

	// 检查 ThinkingEnabled 显式开关
	if cfg.ThinkingEnabled != nil && *cfg.ThinkingEnabled {
		thinkingEnabled = true
	}

	// 检查 ReasoningEffort（OpenAI o-series 格式）
	if cfg.ReasoningEffort != nil {
		effort := *cfg.ReasoningEffort
		if effort != "" && effort != "none" {
			thinkingEnabled = true
		}
	}

	if !thinkingEnabled {
		return systemPrompt
	}

	// 确定 thinking budget
	thinkingBudget := 16000 // 默认保守值
	if cfg.ThinkingTokens != nil && *cfg.ThinkingTokens > 0 {
		thinkingBudget = *cfg.ThinkingTokens
	}

	thinkingHint := fmt.Sprintf("<thinking_mode>enabled</thinking_mode>\n<max_thinking_length>%d</max_thinking_length>", thinkingBudget)
	if systemPrompt != "" {
		return thinkingHint + "\n\n" + systemPrompt
	}
	return thinkingHint
}

// extractToolChoiceHint 从 metadata 中提取 tool_choice 并生成系统提示
// Kiro 不支持原生 tool_choice，通过 prompt hint 注入
func extractToolChoiceHint(metadata map[string]any) string {
	toolChoice, ok := metadata["tool_choice"]
	if !ok || toolChoice == nil {
		return ""
	}

	// string 值: "none", "auto", "required"
	if tc, ok := toolChoice.(string); ok {
		switch tc {
		case "none":
			return "[INSTRUCTION: Do NOT use any tools. Respond with text only.]"
		case "required":
			return "[INSTRUCTION: You MUST use at least one of the available tools to respond. Do not respond with text only - always make a tool call.]"
		case "auto":
			return "" // 默认行为
		}
	}

	// object 值: {"type":"function","function":{"name":"..."}}
	// 先尝试序列化再解析，因为可能是 map 或 struct
	data, err := json.Marshal(toolChoice)
	if err != nil {
		return ""
	}
	var tc struct {
		Type     string `json:"type"`
		Function struct {
			Name string `json:"name"`
		} `json:"function"`
	}
	if err := json.Unmarshal(data, &tc); err == nil && tc.Type == "function" && tc.Function.Name != "" {
		return fmt.Sprintf("[INSTRUCTION: You MUST use the tool named '%s' to respond. Do not use any other tool or respond with text only.]", tc.Function.Name)
	}

	return ""
}

// extractResponseFormatHint 从 metadata 中提取 response_format 并生成系统提示
// Kiro 不支持原生 response_format，通过 prompt hint 注入
func extractResponseFormatHint(metadata map[string]any) string {
	responseFormat, ok := metadata["response_format"]
	if !ok || responseFormat == nil {
		return ""
	}

	data, err := json.Marshal(responseFormat)
	if err != nil {
		return ""
	}
	var rf struct {
		Type       string `json:"type"`
		JSONSchema struct {
			Schema json.RawMessage `json:"schema"`
		} `json:"json_schema"`
	}
	if err := json.Unmarshal(data, &rf); err != nil {
		return ""
	}

	switch rf.Type {
	case "json_object":
		return "[INSTRUCTION: You MUST respond with valid JSON only. Do not include any text before or after the JSON. Do not wrap the JSON in markdown code blocks. Output raw JSON directly.]"
	case "json_schema":
		if len(rf.JSONSchema.Schema) > 0 {
			schemaStr := string(rf.JSONSchema.Schema)
			if len(schemaStr) > 500 {
				schemaStr = schemaStr[:500] + "..."
			}
			return fmt.Sprintf("[INSTRUCTION: You MUST respond with valid JSON that matches this schema: %s. Do not include any text before or after the JSON. Do not wrap the JSON in markdown code blocks. Output raw JSON directly.]", schemaStr)
		}
		return "[INSTRUCTION: You MUST respond with valid JSON only. Do not include any text before or after the JSON. Do not wrap the JSON in markdown code blocks. Output raw JSON directly.]"
	case "text":
		return ""
	}

	return ""
}
