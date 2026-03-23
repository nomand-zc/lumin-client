package converter

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nomand-zc/lumin-client/providers"
)

// 默认安全设置，对齐 CLIProxyAPIPlus 中的 DefaultSafetySettings
var defaultSafetySettings = []map[string]string{
	{"category": "HARM_CATEGORY_HARASSMENT", "threshold": "OFF"},
	{"category": "HARM_CATEGORY_HATE_SPEECH", "threshold": "OFF"},
	{"category": "HARM_CATEGORY_SEXUALLY_EXPLICIT", "threshold": "OFF"},
	{"category": "HARM_CATEGORY_DANGEROUS_CONTENT", "threshold": "OFF"},
	{"category": "HARM_CATEGORY_CIVIC_INTEGRITY", "threshold": "BLOCK_NONE"},
}

// thoughtSignature 常量，对齐 CLIProxyAPIPlus
const thoughtSignatureSkip = "skip_thought_signature_validator"

// ========== Gemini CLI API 请求结构体 ==========

// GeminiCLIRequest 是发送到 Gemini CLI API 的顶层请求结构
// 对齐 gemini-cli converter.ts 中 CAGenerateContentRequest 定义
type GeminiCLIRequest struct {
	Model              string          `json:"model"`
	Project            string          `json:"project,omitempty"`
	UserPromptID       string          `json:"user_prompt_id,omitempty"`
	Request            *GeminiCLIInner `json:"request"`
	EnabledCreditTypes []string        `json:"enabled_credit_types,omitempty"`
}

// GeminiCLIInner 是嵌套的 request 部分
// 对齐 gemini-cli converter.ts 中 VertexGenerateContentRequest 定义
type GeminiCLIInner struct {
	Contents          []GeminiContent         `json:"contents"`
	SystemInstruction *GeminiContent          `json:"systemInstruction,omitempty"`
	CachedContent     string                  `json:"cachedContent,omitempty"`
	Tools             []GeminiTool            `json:"tools,omitempty"`
	ToolConfig        *GeminiToolConfig       `json:"toolConfig,omitempty"`
	Labels            map[string]string       `json:"labels,omitempty"`
	GenerationConfig  *GeminiGenerationConfig `json:"generationConfig,omitempty"`
	SafetySettings    []map[string]string     `json:"safetySettings"`
	SessionID         string                  `json:"session_id,omitempty"`
}

// GeminiContent 代表一条消息
type GeminiContent struct {
	Role  string       `json:"role"`
	Parts []GeminiPart `json:"parts"`
}

// GeminiPart 代表消息中的一个部分
type GeminiPart struct {
	Text               string              `json:"text,omitempty"`
	Thought            *bool               `json:"thought,omitempty"`
	ThoughtSignature   string              `json:"thoughtSignature,omitempty"`
	FunctionCall       *GeminiFunctionCall  `json:"functionCall,omitempty"`
	FunctionResponse   *GeminiFuncResponse  `json:"functionResponse,omitempty"`
	InlineData         *GeminiInlineData    `json:"inlineData,omitempty"`
}

// GeminiFunctionCall 代表函数调用
type GeminiFunctionCall struct {
	Name string         `json:"name"`
	Args map[string]any `json:"args,omitempty"`
}

// GeminiFuncResponse 代表函数响应
type GeminiFuncResponse struct {
	Name     string              `json:"name"`
	Response GeminiFuncRespValue `json:"response"`
}

// GeminiFuncRespValue 代表函数响应的值
type GeminiFuncRespValue struct {
	Result string `json:"result"`
}

// GeminiInlineData 代表内联数据（如图片）
type GeminiInlineData struct {
	MimeType string `json:"mime_type"`
	Data     string `json:"data"`
}

// GeminiTool 代表工具定义
type GeminiTool struct {
	FunctionDeclarations []GeminiFunctionDecl `json:"functionDeclarations,omitempty"`
}

// GeminiFunctionDecl 代表函数声明
type GeminiFunctionDecl struct {
	Name                 string `json:"name"`
	Description          string `json:"description,omitempty"`
	ParametersJsonSchema any    `json:"parametersJsonSchema,omitempty"`
}

// GeminiToolConfig 工具配置
type GeminiToolConfig struct {
	FunctionCallingConfig *GeminiFunctionCallingConfig `json:"functionCallingConfig,omitempty"`
}

// GeminiFunctionCallingConfig 函数调用配置
type GeminiFunctionCallingConfig struct {
	Mode                 string   `json:"mode,omitempty"`
	AllowedFunctionNames []string `json:"allowedFunctionNames,omitempty"`
}

// GeminiGenerationConfig 生成配置
// 对齐 gemini-cli converter.ts 中 VertexGenerationConfig 定义
type GeminiGenerationConfig struct {
	Temperature      *float64              `json:"temperature,omitempty"`
	TopP             *float64              `json:"topP,omitempty"`
	TopK             *float64              `json:"topK,omitempty"`
	MaxOutputTokens  *int                  `json:"maxOutputTokens,omitempty"`
	StopSequences    []string              `json:"stopSequences,omitempty"`
	PresencePenalty  *float64              `json:"presencePenalty,omitempty"`
	FrequencyPenalty *float64              `json:"frequencyPenalty,omitempty"`
	ThinkingConfig   *GeminiThinkingConfig `json:"thinkingConfig,omitempty"`
}

// GeminiThinkingConfig thinking 配置
type GeminiThinkingConfig struct {
	ThinkingBudget  *int  `json:"thinkingBudget,omitempty"`
	ThinkingLevel   string `json:"thinkingLevel,omitempty"`
	IncludeThoughts *bool  `json:"includeThoughts,omitempty"`
}

// ========== 转换逻辑 ==========

// BuildRequest 将统一 Request 转换为 GeminiCLI 请求格式
// 对齐 CLIProxyAPIPlus 中 ConvertClaudeRequestToCLI 的逻辑
func BuildRequest(req *providers.Request, projectID string) (*GeminiCLIRequest, error) {
	if req == nil {
		return nil, fmt.Errorf("request is nil")
	}

	inner := &GeminiCLIInner{
		Contents:       make([]GeminiContent, 0),
		SafetySettings: defaultSafetySettings,
	}

	// 1. 提取 system 消息为 systemInstruction
	// 2. 转换 user/assistant/tool 消息为 contents
	for _, msg := range req.Messages {
		switch msg.Role {
		case providers.RoleSystem:
			// system 消息提取为 systemInstruction，对齐 CLIProxyAPIPlus
			systemParts := make([]GeminiPart, 0)
			if msg.Content != "" {
				systemParts = append(systemParts, GeminiPart{Text: msg.Content})
			}
			// 支持多模态 system 消息
			for _, cp := range msg.ContentParts {
				if cp.Type == providers.ContentTypeText && cp.Text != nil {
					systemParts = append(systemParts, GeminiPart{Text: *cp.Text})
				}
			}
			if len(systemParts) > 0 {
				inner.SystemInstruction = &GeminiContent{
					Role:  "user",
					Parts: systemParts,
				}
			}

		case providers.RoleUser:
			content := convertUserMessage(&msg)
			inner.Contents = append(inner.Contents, content)

		case providers.RoleAssistant:
			content := convertAssistantMessage(&msg)
			inner.Contents = append(inner.Contents, content)

		case providers.RoleTool:
			content := convertToolMessage(&msg)
			inner.Contents = append(inner.Contents, content)
		}
	}

	// 对齐 CLIProxyAPIPlus 中的 fixCLIToolResponse 逻辑：
	// 将分散的 functionResponse 消息合并到对应的 functionCall 之后
	inner.Contents = fixToolResponseGrouping(inner.Contents)

	// 3. 转换 tools 为 functionDeclarations
	if len(req.Tools) > 0 {
		tool := GeminiTool{
			FunctionDeclarations: make([]GeminiFunctionDecl, 0, len(req.Tools)),
		}
		for _, t := range req.Tools {
			decl := GeminiFunctionDecl{
				Name:        t.Name,
				Description: t.Description,
			}
			// 使用 parametersJsonSchema（对齐 CLIProxyAPIPlus）
			if t.Parameters.Type != "" {
				decl.ParametersJsonSchema = t.Parameters
			}
			tool.FunctionDeclarations = append(tool.FunctionDeclarations, decl)
		}
		inner.Tools = []GeminiTool{tool}
	}

	// 4. 处理 tool_choice（从 Metadata 中获取）
	if req.Metadata != nil {
		if toolChoice, ok := req.Metadata["tool_choice"]; ok {
			inner.ToolConfig = convertToolChoice(toolChoice)
		}
	}

	// 5. 映射 GenerationConfig
	genConfig := convertGenerationConfig(&req.GenerationConfig)
	if genConfig != nil {
		inner.GenerationConfig = genConfig
	}

	// 6. 从 Metadata 中提取可选的 user_prompt_id、session_id、enabled_credit_types、labels
	var userPromptID string
	var sessionID string
	var enabledCreditTypes []string
	if req.Metadata != nil {
		if v, ok := req.Metadata["user_prompt_id"].(string); ok {
			userPromptID = v
		}
		if v, ok := req.Metadata["session_id"].(string); ok {
			sessionID = v
		}
		if v, ok := req.Metadata["enabled_credit_types"].([]string); ok {
			enabledCreditTypes = v
		}
		if v, ok := req.Metadata["labels"].(map[string]string); ok {
			inner.Labels = v
		}
		if v, ok := req.Metadata["cached_content"].(string); ok {
			inner.CachedContent = v
		}
	}
	inner.SessionID = sessionID

	return &GeminiCLIRequest{
		Model:              req.Model,
		Project:            projectID,
		UserPromptID:       userPromptID,
		Request:            inner,
		EnabledCreditTypes: enabledCreditTypes,
	}, nil
}

// convertUserMessage 将 user 消息转换为 GeminiContent
func convertUserMessage(msg *providers.Message) GeminiContent {
	content := GeminiContent{
		Role:  "user",
		Parts: make([]GeminiPart, 0),
	}

	// 文本内容
	if msg.Content != "" {
		content.Parts = append(content.Parts, GeminiPart{Text: msg.Content})
	}

	// 多模态内容
	for _, cp := range msg.ContentParts {
		switch cp.Type {
		case providers.ContentTypeText:
			if cp.Text != nil {
				content.Parts = append(content.Parts, GeminiPart{Text: *cp.Text})
			}
		case providers.ContentTypeImage:
			if cp.Image != nil {
				part := convertImagePart(cp.Image)
				if part != nil {
					content.Parts = append(content.Parts, *part)
				}
			}
		}
	}

	return content
}

// convertAssistantMessage 将 assistant 消息转换为 GeminiContent（role=model）
func convertAssistantMessage(msg *providers.Message) GeminiContent {
	content := GeminiContent{
		Role:  "model",
		Parts: make([]GeminiPart, 0),
	}

	// thinking/reasoning 内容
	if msg.ReasoningContent != "" {
		isThought := true
		content.Parts = append(content.Parts, GeminiPart{
			Text:    msg.ReasoningContent,
			Thought: &isThought,
		})
	}

	// 文本内容
	if msg.Content != "" {
		content.Parts = append(content.Parts, GeminiPart{Text: msg.Content})
	}

	// tool calls 转换为 functionCall
	for _, tc := range msg.ToolCalls {
		fc := &GeminiFunctionCall{
			Name: tc.Function.Name,
		}
		// 解析 arguments
		if len(tc.Function.Arguments) > 0 {
			var args map[string]any
			if err := json.Unmarshal(tc.Function.Arguments, &args); err == nil {
				fc.Args = args
			}
		}
		part := GeminiPart{
			ThoughtSignature: thoughtSignatureSkip,
			FunctionCall:     fc,
		}
		// 如果有 ExtraFields 中的 thoughtSignature，使用它
		if tc.ExtraFields != nil {
			if sig, ok := tc.ExtraFields["thoughtSignature"].(string); ok && sig != "" {
				part.ThoughtSignature = sig
			}
		}
		content.Parts = append(content.Parts, part)
	}

	return content
}

// convertToolMessage 将 tool 消息转换为 GeminiContent（role=function）
func convertToolMessage(msg *providers.Message) GeminiContent {
	funcName := msg.ToolName
	if funcName == "" && msg.ToolID != "" {
		// 尝试从 ToolID 中提取函数名（对齐 CLIProxyAPIPlus 的逻辑）
		parts := strings.Split(msg.ToolID, "-")
		if len(parts) > 1 {
			funcName = strings.Join(parts[0:len(parts)-1], "-")
		} else {
			funcName = msg.ToolID
		}
	}

	return GeminiContent{
		Role: "function",
		Parts: []GeminiPart{
			{
				FunctionResponse: &GeminiFuncResponse{
					Name: funcName,
					Response: GeminiFuncRespValue{
						Result: msg.Content,
					},
				},
			},
		},
	}
}

// convertImagePart 将图片数据转换为 GeminiPart
func convertImagePart(img *providers.Image) *GeminiPart {
	if img == nil {
		return nil
	}

	// 如果有 URL，目前 Gemini CLI 不直接支持 URL，跳过
	// 如果有 Data，使用 inlineData
	if len(img.Data) > 0 {
		mimeType := "image/png"
		switch img.Format {
		case "jpg", "jpeg":
			mimeType = "image/jpeg"
		case "webp":
			mimeType = "image/webp"
		case "gif":
			mimeType = "image/gif"
		case "png":
			mimeType = "image/png"
		}
		return &GeminiPart{
			InlineData: &GeminiInlineData{
				MimeType: mimeType,
				Data:     base64.StdEncoding.EncodeToString(img.Data),
			},
		}
	}

	return nil
}

// convertToolChoice 将 tool_choice 转换为 GeminiToolConfig
func convertToolChoice(toolChoice any) *GeminiToolConfig {
	config := &GeminiToolConfig{
		FunctionCallingConfig: &GeminiFunctionCallingConfig{},
	}

	switch v := toolChoice.(type) {
	case string:
		switch v {
		case "auto":
			config.FunctionCallingConfig.Mode = "AUTO"
		case "none":
			config.FunctionCallingConfig.Mode = "NONE"
		case "any", "required":
			config.FunctionCallingConfig.Mode = "ANY"
		default:
			return nil
		}
	case map[string]any:
		tcType, _ := v["type"].(string)
		tcName, _ := v["name"].(string)
		switch tcType {
		case "auto":
			config.FunctionCallingConfig.Mode = "AUTO"
		case "none":
			config.FunctionCallingConfig.Mode = "NONE"
		case "any":
			config.FunctionCallingConfig.Mode = "ANY"
		case "tool", "function":
			config.FunctionCallingConfig.Mode = "ANY"
			if tcName != "" {
				config.FunctionCallingConfig.AllowedFunctionNames = []string{tcName}
			}
		default:
			return nil
		}
	default:
		return nil
	}

	return config
}

// convertGenerationConfig 将统一 GenerationConfig 转换为 Gemini 格式
func convertGenerationConfig(cfg *providers.GenerationConfig) *GeminiGenerationConfig {
	if cfg == nil {
		return nil
	}

	genCfg := &GeminiGenerationConfig{}
	hasConfig := false

	if cfg.Temperature != nil {
		genCfg.Temperature = cfg.Temperature
		hasConfig = true
	}
	if cfg.TopP != nil {
		genCfg.TopP = cfg.TopP
		hasConfig = true
	}
	if cfg.PresencePenalty != nil {
		genCfg.PresencePenalty = cfg.PresencePenalty
		hasConfig = true
	}
	if cfg.FrequencyPenalty != nil {
		genCfg.FrequencyPenalty = cfg.FrequencyPenalty
		hasConfig = true
	}
	if cfg.MaxTokens != nil {
		genCfg.MaxOutputTokens = cfg.MaxTokens
		hasConfig = true
	}
	if len(cfg.Stop) > 0 {
		genCfg.StopSequences = cfg.Stop
		hasConfig = true
	}

	// thinking 配置
	if cfg.ThinkingEnabled != nil {
		include := *cfg.ThinkingEnabled
		thinkingCfg := &GeminiThinkingConfig{
			IncludeThoughts: &include,
		}
		if cfg.ThinkingTokens != nil {
			budget := *cfg.ThinkingTokens
			thinkingCfg.ThinkingBudget = &budget
		}
		genCfg.ThinkingConfig = thinkingCfg
		hasConfig = true
	}

	// reasoning effort 映射为 thinkingLevel
	if cfg.ReasoningEffort != nil && *cfg.ReasoningEffort != "" {
		effort := strings.ToLower(*cfg.ReasoningEffort)
		if genCfg.ThinkingConfig == nil {
			genCfg.ThinkingConfig = &GeminiThinkingConfig{}
		}
		genCfg.ThinkingConfig.ThinkingLevel = effort
		include := effort != "none"
		genCfg.ThinkingConfig.IncludeThoughts = &include
		hasConfig = true
	}

	if !hasConfig {
		return nil
	}
	return genCfg
}

// fixToolResponseGrouping 对齐 CLIProxyAPIPlus 中 fixCLIToolResponse 的逻辑：
// 将分散的 function role 消息正确地分组到对应的 model（含 functionCall）消息之后
func fixToolResponseGrouping(contents []GeminiContent) []GeminiContent {
	type functionCallGroup struct {
		responsesNeeded int
	}

	result := make([]GeminiContent, 0, len(contents))
	var pendingGroups []*functionCallGroup
	var collectedResponses []GeminiPart

	for _, content := range contents {
		// 收集 function 角色的响应 parts
		if content.Role == "function" {
			for _, part := range content.Parts {
				if part.FunctionResponse != nil {
					collectedResponses = append(collectedResponses, part)
				}
			}
			// 尝试用收集到的响应填充 pending groups
			for i := len(pendingGroups) - 1; i >= 0; i-- {
				group := pendingGroups[i]
				if len(collectedResponses) >= group.responsesNeeded {
					groupResponses := collectedResponses[:group.responsesNeeded]
					collectedResponses = collectedResponses[group.responsesNeeded:]
					funcContent := GeminiContent{
						Role:  "function",
						Parts: groupResponses,
					}
					result = append(result, funcContent)
					pendingGroups = append(pendingGroups[:i], pendingGroups[i+1:]...)
					break
				}
			}
			continue
		}

		// model 角色的消息：检查是否包含 functionCall
		if content.Role == "model" {
			functionCallCount := 0
			for _, part := range content.Parts {
				if part.FunctionCall != nil {
					functionCallCount++
				}
			}
			result = append(result, content)
			if functionCallCount > 0 {
				pendingGroups = append(pendingGroups, &functionCallGroup{
					responsesNeeded: functionCallCount,
				})
			}
			continue
		}

		// 其他角色直接添加
		result = append(result, content)
	}

	// 处理剩余的 pending groups
	for _, group := range pendingGroups {
		if len(collectedResponses) >= group.responsesNeeded {
			groupResponses := collectedResponses[:group.responsesNeeded]
			collectedResponses = collectedResponses[group.responsesNeeded:]
			funcContent := GeminiContent{
				Role:  "function",
				Parts: groupResponses,
			}
			result = append(result, funcContent)
		}
	}

	return result
}
