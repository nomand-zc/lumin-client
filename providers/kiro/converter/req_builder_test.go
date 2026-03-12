package converter

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/nomand-zc/lumin-client/providers"
)

// ============================================================
// 辅助函数
// ============================================================

func strPtr(s string) *string {
	return &s
}

// makeTextPart 构造文本类型的 ContentPart
func makeTextPart(text string) providers.ContentPart {
	return providers.ContentPart{
		Type: providers.ContentTypeText,
		Text: strPtr(text),
	}
}

// makeImagePart 构造图片类型的 ContentPart
func makeImagePart(data []byte, format string) providers.ContentPart {
	return providers.ContentPart{
		Type: providers.ContentTypeImage,
		Image: &providers.Image{
			Data:   data,
			Format: format,
		},
	}
}

// makeTool 快速构造测试工具
func makeTool(name, desc string) providers.Tool {
	return providers.Tool{
		Name:        name,
		Description: desc,
		Parameters: providers.Schema{
			Type:       "object",
			Properties: map[string]*providers.Schema{},
		},
	}
}

// makeUserMsg 快速构造 user 消息（Content 字段）
func makeUserMsg(content string) providers.Message {
	return providers.Message{Role: providers.RoleUser, Content: content}
}

// makeAssistantMsg 快速构造 assistant 消息（Content 字段）
func makeAssistantMsg(content string) providers.Message {
	return providers.Message{Role: providers.RoleAssistant, Content: content}
}

// makeSystemMsg 快速构造 system 消息
func makeSystemMsg(content string) providers.Message {
	return providers.Message{Role: providers.RoleSystem, Content: content}
}

// makeToolMsg 快速构造 tool 消息
func makeToolMsg(toolID, content string) providers.Message {
	return providers.Message{Role: providers.RoleTool, ToolID: toolID, Content: content}
}

// newReq 快速构造 providers.Request 指针
func newReq(model string, messages ...providers.Message) *providers.Request {
	return &providers.Request{
		Model:    model,
		Messages: messages,
	}
}

// ============================================================
// 任务 2：空输入与 nil 返回测试
// ============================================================

// TestConvertRequest_EmptyMessages 传入空消息列表，应返回 error
func TestConvertRequest_EmptyMessages(t *testing.T) {
	req := &providers.Request{Model: "claude-sonnet-4.5", Messages: []providers.Message{}}
	result, err := ConvertRequest(context.Background(), req)
	if err == nil {
		t.Error("期望返回 error，实际 error 为 nil")
	}
	if result != nil {
		t.Errorf("期望返回 nil，实际返回 %+v", result)
	}
}

// TestConvertRequest_OnlySystemMessages 仅含 system 消息，应返回 (nil, nil)（正常提前退出）
func TestConvertRequest_OnlySystemMessages(t *testing.T) {
	req := newReq("claude-sonnet-4.5", makeSystemMsg("你是一个助手"))
	result, err := ConvertRequest(context.Background(), req)
	if err != nil {
		t.Errorf("期望 error 为 nil，实际 %v", err)
	}
	if result != nil {
		t.Errorf("期望返回 nil，实际返回 %+v", result)
	}
}

// TestConvertRequest_LastAssistantIsBrace_BecomesEmpty 唯一消息为 content="{" 的 assistant 消息，预处理后为空，应返回 (nil, nil)
func TestConvertRequest_LastAssistantIsBrace_BecomesEmpty(t *testing.T) {
	req := newReq("claude-sonnet-4.5", makeAssistantMsg("{"))
	result, err := ConvertRequest(context.Background(), req)
	if err != nil {
		t.Errorf("期望 error 为 nil，实际 %v", err)
	}
	if result != nil {
		t.Errorf("期望返回 nil，实际返回 %+v", result)
	}
}

// ============================================================
// 任务 3：基础请求结构与模型 ID 解析测试
// ============================================================

// TestConvertRequest_BasicStructure 传入单条 user 消息，验证基础结构字段
func TestConvertRequest_BasicStructure(t *testing.T) {
	req := newReq("claude-sonnet-4.5", makeUserMsg("hello"))
	result, err := ConvertRequest(context.Background(), req)
	if err != nil {
		t.Fatalf("期望 error 为 nil，实际 %v", err)
	}
	if result == nil {
		t.Fatal("期望返回非 nil 结果")
	}
	if result.ConversationState.ChatTriggerType != "MANUAL" {
		t.Errorf("ChatTriggerType 期望 MANUAL，实际 %s", result.ConversationState.ChatTriggerType)
	}
	if result.ConversationState.ConversationId == "" {
		t.Error("ConversationId 不应为空")
	}
	if result.ConversationState.CurrentMessage.UserInputMessage.Origin != "AI_EDITOR" {
		t.Errorf("Origin 期望 AI_EDITOR，实际 %s", result.ConversationState.CurrentMessage.UserInputMessage.Origin)
	}
}

// ============================================================
// 任务 4：消息预处理（preprocessMessages）测试
// ============================================================

// TestPreprocessMessages_RemoveLastBraceAssistant_Content 最后一条 assistant Content="{" 应被移除
func TestPreprocessMessages_RemoveLastBraceAssistant_Content(t *testing.T) {
	msgs := []providers.Message{
		makeUserMsg("hello"),
		makeAssistantMsg("{"),
	}
	result := preprocessMessages(msgs)
	if len(result) != 1 {
		t.Fatalf("期望 1 条消息，实际 %d 条", len(result))
	}
	if result[0].Role != providers.RoleUser {
		t.Errorf("期望保留 user 消息，实际 role=%s", result[0].Role)
	}
}

// TestPreprocessMessages_RemoveLastBraceAssistant_ContentParts 最后一条 assistant ContentParts[0].Text="{" 应被移除
func TestPreprocessMessages_RemoveLastBraceAssistant_ContentParts(t *testing.T) {
	msgs := []providers.Message{
		makeUserMsg("hello"),
		{
			Role:         providers.RoleAssistant,
			ContentParts: []providers.ContentPart{makeTextPart("{")},
		},
	}
	result := preprocessMessages(msgs)
	if len(result) != 1 {
		t.Fatalf("期望 1 条消息，实际 %d 条", len(result))
	}
	if result[0].Role != providers.RoleUser {
		t.Errorf("期望保留 user 消息，实际 role=%s", result[0].Role)
	}
}

// TestPreprocessMessages_KeepNonBraceAssistant 最后一条 assistant 内容不为 "{" 应保留
func TestPreprocessMessages_KeepNonBraceAssistant(t *testing.T) {
	msgs := []providers.Message{
		makeUserMsg("hello"),
		makeAssistantMsg("world"),
	}
	result := preprocessMessages(msgs)
	if len(result) != 2 {
		t.Fatalf("期望 2 条消息，实际 %d 条", len(result))
	}
	if result[1].Content != "world" {
		t.Errorf("期望 assistant 内容为 world，实际 %s", result[1].Content)
	}
}

// TestPreprocessMessages_MergeAdjacentUser_BothContent 两条相邻 user 消息均用 Content，应合并
func TestPreprocessMessages_MergeAdjacentUser_BothContent(t *testing.T) {
	msgs := []providers.Message{
		makeUserMsg("hello"),
		makeUserMsg("world"),
	}
	result := preprocessMessages(msgs)
	if len(result) != 1 {
		t.Fatalf("期望合并为 1 条消息，实际 %d 条", len(result))
	}
	expected := "hello\nworld"
	if result[0].Content != expected {
		t.Errorf("期望内容 %q，实际 %q", expected, result[0].Content)
	}
}

// TestPreprocessMessages_MergeAdjacentUser_BothContentParts 两条相邻 user 消息均用 ContentParts，应追加合并
func TestPreprocessMessages_MergeAdjacentUser_BothContentParts(t *testing.T) {
	msgs := []providers.Message{
		{Role: providers.RoleUser, ContentParts: []providers.ContentPart{makeTextPart("part1")}},
		{Role: providers.RoleUser, ContentParts: []providers.ContentPart{makeTextPart("part2")}},
	}
	result := preprocessMessages(msgs)
	if len(result) != 1 {
		t.Fatalf("期望合并为 1 条消息，实际 %d 条", len(result))
	}
	if len(result[0].ContentParts) != 2 {
		t.Fatalf("期望 2 个 ContentParts，实际 %d 个", len(result[0].ContentParts))
	}
	if result[0].ContentParts[0].Text == nil || *result[0].ContentParts[0].Text != "part1" {
		t.Error("第一个 ContentPart 内容不正确")
	}
	if result[0].ContentParts[1].Text == nil || *result[0].ContentParts[1].Text != "part2" {
		t.Error("第二个 ContentPart 内容不正确")
	}
}

// TestPreprocessMessages_MergeAdjacentUser_PrevPartsNextContent 前者 ContentParts、后者 Content，后者应追加为 text part
func TestPreprocessMessages_MergeAdjacentUser_PrevPartsNextContent(t *testing.T) {
	msgs := []providers.Message{
		{Role: providers.RoleUser, ContentParts: []providers.ContentPart{makeTextPart("part1")}},
		makeUserMsg("content2"),
	}
	result := preprocessMessages(msgs)
	if len(result) != 1 {
		t.Fatalf("期望合并为 1 条消息，实际 %d 条", len(result))
	}
	if len(result[0].ContentParts) != 2 {
		t.Fatalf("期望 2 个 ContentParts，实际 %d 个", len(result[0].ContentParts))
	}
	if result[0].ContentParts[1].Text == nil || *result[0].ContentParts[1].Text != "content2" {
		t.Error("追加的 text part 内容不正确")
	}
}

// TestPreprocessMessages_MergeAdjacentUser_PrevContentNextParts 前者 Content、后者 ContentParts，前者应转换为 ContentParts 后追加
func TestPreprocessMessages_MergeAdjacentUser_PrevContentNextParts(t *testing.T) {
	msgs := []providers.Message{
		makeUserMsg("content1"),
		{Role: providers.RoleUser, ContentParts: []providers.ContentPart{makeTextPart("part2")}},
	}
	result := preprocessMessages(msgs)
	if len(result) != 1 {
		t.Fatalf("期望合并为 1 条消息，实际 %d 条", len(result))
	}
	if result[0].Content != "" {
		t.Errorf("合并后 Content 应为空，实际 %q", result[0].Content)
	}
	if len(result[0].ContentParts) != 2 {
		t.Fatalf("期望 2 个 ContentParts，实际 %d 个", len(result[0].ContentParts))
	}
	if result[0].ContentParts[0].Text == nil || *result[0].ContentParts[0].Text != "content1" {
		t.Error("第一个 ContentPart 内容不正确，期望 content1")
	}
	if result[0].ContentParts[1].Text == nil || *result[0].ContentParts[1].Text != "part2" {
		t.Error("第二个 ContentPart 内容不正确，期望 part2")
	}
}

// ============================================================
// 任务 5：System Prompt 处理测试
// ============================================================

// TestConvertRequest_SystemPrompt_MergedWithFirstUser system + user + assistant + user，system 应与第一条 user 合并
// 注意：相邻两条 user 消息会被 preprocessMessages 合并，所以需要用 user+assistant+user 来测试 system 与第一条 user 合并的场景
func TestConvertRequest_SystemPrompt_MergedWithFirstUser(t *testing.T) {
	req := newReq("claude-sonnet-4.5",
		makeSystemMsg("系统提示"),
		makeUserMsg("用户消息1"),
		makeAssistantMsg("助手回复"),
		makeUserMsg("用户消息2"),
	)
	result, err := ConvertRequest(context.Background(), req)
	if err != nil {
		t.Fatalf("期望 error 为 nil，实际 %v", err)
	}
	if result == nil {
		t.Fatal("期望返回非 nil 结果")
	}
	history := result.ConversationState.History
	if len(history) == 0 {
		t.Fatal("期望 history 非空")
	}
	firstHistoryItem := history[0]
	if firstHistoryItem.UserInputMessage == nil {
		t.Fatal("期望 history[0] 为 UserInputMessage")
	}
	// system 与第一条 user 消息合并
	expected := "系统提示\n\n用户消息1"
	if firstHistoryItem.UserInputMessage.Content != expected {
		t.Errorf("期望 history[0] 内容为 %q，实际 %q", expected, firstHistoryItem.UserInputMessage.Content)
	}
}

// TestConvertRequest_SystemPrompt_StandaloneBeforeAssistant system + assistant + user，system 应单独作为 history[0]
func TestConvertRequest_SystemPrompt_StandaloneBeforeAssistant(t *testing.T) {
	req := newReq("claude-sonnet-4.5",
		makeSystemMsg("系统提示"),
		makeAssistantMsg("助手回复"),
		makeUserMsg("用户消息"),
	)
	result, err := ConvertRequest(context.Background(), req)
	if err != nil {
		t.Fatalf("期望 error 为 nil，实际 %v", err)
	}
	if result == nil {
		t.Fatal("期望返回非 nil 结果")
	}
	history := result.ConversationState.History
	if len(history) == 0 {
		t.Fatal("期望 history 非空")
	}
	firstHistoryItem := history[0]
	if firstHistoryItem.UserInputMessage == nil {
		t.Fatal("期望 history[0] 为 UserInputMessage")
	}
	if firstHistoryItem.UserInputMessage.Content != "系统提示" {
		t.Errorf("期望 history[0] 内容为 %q，实际 %q", "系统提示", firstHistoryItem.UserInputMessage.Content)
	}
}

// TestConvertRequest_MultipleSystemPrompts 两条 system 消息，应合并后再与第一条 user 消息合并
// 注意：preprocessMessages 会将相邻 system 消息用 "\n" 合并，然后 ConvertRequest 再将 system prompt 与 user 消息用 "\n\n" 合并
func TestConvertRequest_MultipleSystemPrompts(t *testing.T) {
	req := newReq("claude-sonnet-4.5",
		makeSystemMsg("系统提示1"),
		makeSystemMsg("系统提示2"),
		makeUserMsg("用户消息"),
	)
	result, err := ConvertRequest(context.Background(), req)
	if err != nil {
		t.Fatalf("期望 error 为 nil，实际 %v", err)
	}
	if result == nil {
		t.Fatal("期望返回非 nil 结果")
	}
	history := result.ConversationState.History
	if len(history) == 0 {
		t.Fatal("期望 history 非空")
	}
	firstHistoryItem := history[0]
	if firstHistoryItem.UserInputMessage == nil {
		t.Fatal("期望 history[0] 为 UserInputMessage")
	}
	// preprocessMessages 将两条 system 消息用 "\n" 合并为一条，
	// ConvertRequest 提取 system prompt 后再与 user 消息用 "\n\n" 合并
	expected := "系统提示1\n系统提示2\n\n用户消息"
	if firstHistoryItem.UserInputMessage.Content != expected {
		t.Errorf("期望 history[0] 内容为 %q，实际 %q", expected, firstHistoryItem.UserInputMessage.Content)
	}
}

// TestConvertRequest_EmptySystemPrompt_Ignored system 消息 Content 为空，应被忽略
func TestConvertRequest_EmptySystemPrompt_Ignored(t *testing.T) {
	req := newReq("claude-sonnet-4.5",
		makeSystemMsg(""),
		makeUserMsg("用户消息"),
	)
	result, err := ConvertRequest(context.Background(), req)
	if err != nil {
		t.Fatalf("期望 error 为 nil，实际 %v", err)
	}
	if result == nil {
		t.Fatal("期望返回非 nil 结果")
	}
	// 空 system 消息被忽略，只有一条 user 消息，history 应为空
	history := result.ConversationState.History
	if len(history) != 0 {
		t.Errorf("期望 history 为空，实际 %d 条", len(history))
	}
	// currentMessage 应为用户消息内容
	currentContent := result.ConversationState.CurrentMessage.UserInputMessage.Content
	if currentContent != "用户消息" {
		t.Errorf("期望 currentContent=%q，实际=%q", "用户消息", currentContent)
	}
}

// ============================================================
// 任务 6：History 构建与 Content 兜底测试
// ============================================================

// TestConvertRequest_SingleUserMessage_EmptyHistory 单条 user 消息，history 应为空
func TestConvertRequest_SingleUserMessage_EmptyHistory(t *testing.T) {
	req := newReq("claude-sonnet-4.5", makeUserMsg("hello"))
	result, err := ConvertRequest(context.Background(), req)
	if err != nil {
		t.Fatalf("期望 error 为 nil，实际 %v", err)
	}
	if result == nil {
		t.Fatal("期望返回非 nil 结果")
	}
	if len(result.ConversationState.History) != 0 {
		t.Errorf("期望 history 为空，实际 %d 条", len(result.ConversationState.History))
	}
}

// TestConvertRequest_LastAssistantMovedToHistory 最后一条为 assistant 消息，应进入 history，currentContent 为 "Continue"
func TestConvertRequest_LastAssistantMovedToHistory(t *testing.T) {
	req := newReq("claude-sonnet-4.5",
		makeUserMsg("用户消息"),
		makeAssistantMsg("助手回复"),
	)
	result, err := ConvertRequest(context.Background(), req)
	if err != nil {
		t.Fatalf("期望 error 为 nil，实际 %v", err)
	}
	if result == nil {
		t.Fatal("期望返回非 nil 结果")
	}
	history := result.ConversationState.History
	// history 应包含 user 消息和 assistant 消息
	if len(history) < 2 {
		t.Fatalf("期望 history 至少 2 条，实际 %d 条", len(history))
	}
	// 最后一条 history 应为 assistant 消息
	lastHistoryItem := history[len(history)-1]
	if lastHistoryItem.AssistantResponseMessage == nil {
		t.Error("期望 history 最后一条为 AssistantResponseMessage")
	}
	if lastHistoryItem.AssistantResponseMessage.Content != "助手回复" {
		t.Errorf("期望 assistant 内容为 %q，实际 %q", "助手回复", lastHistoryItem.AssistantResponseMessage.Content)
	}
	// currentContent 应为 "Continue"
	currentContent := result.ConversationState.CurrentMessage.UserInputMessage.Content
	if currentContent != "Continue" {
		t.Errorf("期望 currentContent=%q，实际=%q", "Continue", currentContent)
	}
}

// TestConvertRequest_AutoInsertContinueInHistory history 末尾为 user 消息时，应自动插入 AssistantResponseMessage{Content:"Continue"}
// 构造场景：user + assistant + user（history 中有 user 消息，且末尾为 user 消息），最后一条 user 消息作为 currentMessage
// 此时 history 末尾是 user 消息，应自动插入 AssistantResponseMessage{Content:"Continue"}
func TestConvertRequest_AutoInsertContinueInHistory(t *testing.T) {
	// 构造：system + user + user + assistant + user
	// preprocessMessages 后：system + user(合并) + assistant + user
	// 去掉 system 后：user(合并) + assistant + user
	// history 包含：user(合并) + assistant，末尾为 assistant，不触发自动插入
	// 需要构造 history 末尾为 user 的场景：
	// user + tool + user（tool 消息在 history 中作为 UserInputMessage，末尾为 user 类型）
	req := newReq("claude-sonnet-4.5",
		makeUserMsg("第一条用户消息"),
		makeAssistantMsg("助手回复"),
		makeToolMsg("tool-id", "工具结果"),
		makeUserMsg("最后用户消息"),
	)
	result, err := ConvertRequest(context.Background(), req)
	if err != nil {
		t.Fatalf("期望 error 为 nil，实际 %v", err)
	}
	if result == nil {
		t.Fatal("期望返回非 nil 结果")
	}
	history := result.ConversationState.History
	if len(history) == 0 {
		t.Fatal("期望 history 非空")
	}
	// history 末尾应为 AssistantResponseMessage{Content:"Continue"}（因为 tool 消息作为 UserInputMessage 在 history 末尾）
	lastHistoryItem := history[len(history)-1]
	if lastHistoryItem.AssistantResponseMessage == nil {
		t.Errorf("期望 history 末尾为 AssistantResponseMessage，实际 %+v", lastHistoryItem)
	} else if lastHistoryItem.AssistantResponseMessage.Content != "Continue" {
		t.Errorf("期望自动插入的 assistant 内容为 Continue，实际 %q", lastHistoryItem.AssistantResponseMessage.Content)
	}
}

// TestConvertRequest_EmptyUserContent_Fallback 最后一条 user 消息 content 为空且无 toolResults，currentContent 应兜底为 "Continue"
func TestConvertRequest_EmptyUserContent_Fallback(t *testing.T) {
	req := newReq("claude-sonnet-4.5",
		makeUserMsg(""),
	)
	result, err := ConvertRequest(context.Background(), req)
	if err != nil {
		t.Fatalf("期望 error 为 nil，实际 %v", err)
	}
	if result == nil {
		t.Fatal("期望返回非 nil 结果")
	}
	currentContent := result.ConversationState.CurrentMessage.UserInputMessage.Content
	if currentContent != "Continue" {
		t.Errorf("期望 currentContent=%q，实际=%q", "Continue", currentContent)
	}
}

// TestConvertRequest_EmptyUserContent_WithToolResults_Fallback 最后一条 user 消息 content 为空但有 toolResults，currentContent 应兜底为 "Tool results provided."
func TestConvertRequest_EmptyUserContent_WithToolResults_Fallback(t *testing.T) {
	req := newReq("claude-sonnet-4.5",
		makeUserMsg("用户消息"),
		makeAssistantMsg("助手回复"),
		makeToolMsg("tool-id-1", "工具结果"),
	)
	result, err := ConvertRequest(context.Background(), req)
	if err != nil {
		t.Fatalf("期望 error 为 nil，实际 %v", err)
	}
	if result == nil {
		t.Fatal("期望返回非 nil 结果")
	}
	currentContent := result.ConversationState.CurrentMessage.UserInputMessage.Content
	if currentContent != "Tool results provided." {
		t.Errorf("期望 currentContent=%q，实际=%q", "Tool results provided.", currentContent)
	}
}

// ============================================================
// 任务 7：图片处理（convertImage）与图片阈值测试
// ============================================================

// TestConvertImage_WithData Image.Data 非空，应返回 base64 编码结果
func TestConvertImage_WithData(t *testing.T) {
	data := []byte("fake image data")
	img := &providers.Image{Data: data, Format: "png"}
	result := convertImage(img)
	if result == nil {
		t.Fatal("期望返回非 nil 结果")
	}
	expected := base64.StdEncoding.EncodeToString(data)
	if result.Source.Bytes != expected {
		t.Errorf("期望 base64 编码 %q，实际 %q", expected, result.Source.Bytes)
	}
	if result.Format != "png" {
		t.Errorf("期望 format=png，实际 %s", result.Format)
	}
}

// TestConvertImage_DefaultFormat Image.Format 为空，应默认使用 "jpeg"
func TestConvertImage_DefaultFormat(t *testing.T) {
	data := []byte("fake image data")
	img := &providers.Image{Data: data, Format: ""}
	result := convertImage(img)
	if result == nil {
		t.Fatal("期望返回非 nil 结果")
	}
	if result.Format != "jpeg" {
		t.Errorf("期望默认 format=jpeg，实际 %s", result.Format)
	}
}

// TestConvertImage_URLOnly Image.URL 非空但 Data 为空，应返回 nil
func TestConvertImage_URLOnly(t *testing.T) {
	img := &providers.Image{URL: "https://example.com/image.png", Data: nil}
	result := convertImage(img)
	if result != nil {
		t.Errorf("期望返回 nil（URL 格式不支持），实际 %+v", result)
	}
}

// TestConvertImage_Nil 传入 nil，应返回 nil
func TestConvertImage_Nil(t *testing.T) {
	result := convertImage(nil)
	if result != nil {
		t.Errorf("期望返回 nil，实际 %+v", result)
	}
}

// TestConvertRequest_ImageInCurrentMessage 最后一条 user 消息含图片 ContentPart，图片应放入 UserInputMessage.Images
func TestConvertRequest_ImageInCurrentMessage(t *testing.T) {
	imgData := []byte("image bytes")
	req := &providers.Request{
		Model: "claude-sonnet-4.5",
		Messages: []providers.Message{
			{
				Role:         providers.RoleUser,
				ContentParts: []providers.ContentPart{makeImagePart(imgData, "png")},
			},
		},
	}
	result, err := ConvertRequest(context.Background(), req)
	if err != nil {
		t.Fatalf("期望 error 为 nil，实际 %v", err)
	}
	if result == nil {
		t.Fatal("期望返回非 nil 结果")
	}
	images := result.ConversationState.CurrentMessage.UserInputMessage.Images
	if len(images) != 1 {
		t.Fatalf("期望 1 张图片，实际 %d 张", len(images))
	}
	expected := base64.StdEncoding.EncodeToString(imgData)
	if images[0].Source.Bytes != expected {
		t.Errorf("图片 base64 不匹配")
	}
}

// TestConvertRequest_ImageThreshold_TooFar 图片消息距末尾超过 5 条，图片应被替换为占位符文本
func TestConvertRequest_ImageThreshold_TooFar(t *testing.T) {
	imgData := []byte("image bytes")
	// 构造：图片消息 + 6 条普通消息（交替 user/assistant），最后一条 user
	messages := []providers.Message{
		{
			Role:         providers.RoleUser,
			ContentParts: []providers.ContentPart{makeImagePart(imgData, "png")},
		},
		makeAssistantMsg("回复1"),
		makeUserMsg("消息2"),
		makeAssistantMsg("回复2"),
		makeUserMsg("消息3"),
		makeAssistantMsg("回复3"),
		makeUserMsg("最后消息"),
	}
	req := &providers.Request{Model: "claude-sonnet-4.5", Messages: messages}
	result, err := ConvertRequest(context.Background(), req)
	if err != nil {
		t.Fatalf("期望 error 为 nil，实际 %v", err)
	}
	if result == nil {
		t.Fatal("期望返回非 nil 结果")
	}
	// 图片消息在 history 中，距末尾超过 5，应被替换为占位符
	history := result.ConversationState.History
	if len(history) == 0 {
		t.Fatal("期望 history 非空")
	}
	// 找到第一条 UserInputMessage（图片消息）
	var firstUserMsg *UserInputMessage
	for _, item := range history {
		if item.UserInputMessage != nil {
			firstUserMsg = item.UserInputMessage
			break
		}
	}
	if firstUserMsg == nil {
		t.Fatal("未找到 history 中的 UserInputMessage")
	}
	if len(firstUserMsg.Images) > 0 {
		t.Error("期望图片被替换为占位符，但仍存在 Images")
	}
	if !strings.Contains(firstUserMsg.Content, "图片") {
		t.Errorf("期望 Content 包含图片占位符，实际 %q", firstUserMsg.Content)
	}
}

// TestConvertRequest_ImageThreshold_Close 图片消息距末尾不超过 5 条，图片应被保留
func TestConvertRequest_ImageThreshold_Close(t *testing.T) {
	imgData := []byte("image bytes")
	// 构造：图片消息 + 3 条普通消息，最后一条 user
	messages := []providers.Message{
		{
			Role:         providers.RoleUser,
			ContentParts: []providers.ContentPart{makeImagePart(imgData, "png")},
		},
		makeAssistantMsg("回复1"),
		makeUserMsg("消息2"),
		makeAssistantMsg("回复2"),
		makeUserMsg("最后消息"),
	}
	req := &providers.Request{Model: "claude-sonnet-4.5", Messages: messages}
	result, err := ConvertRequest(context.Background(), req)
	if err != nil {
		t.Fatalf("期望 error 为 nil，实际 %v", err)
	}
	if result == nil {
		t.Fatal("期望返回非 nil 结果")
	}
	history := result.ConversationState.History
	if len(history) == 0 {
		t.Fatal("期望 history 非空")
	}
	// 找到第一条 UserInputMessage（图片消息）
	var firstUserMsg *UserInputMessage
	for _, item := range history {
		if item.UserInputMessage != nil {
			firstUserMsg = item.UserInputMessage
			break
		}
	}
	if firstUserMsg == nil {
		t.Fatal("未找到 history 中的 UserInputMessage")
	}
	if len(firstUserMsg.Images) == 0 {
		t.Error("期望图片被保留，但 Images 为空")
	}
}

// ============================================================
// 任务 8：工具列表处理（buildKiroTools）测试
// ============================================================

// TestBuildKiroTools_EmptyTools 传入空工具列表，应返回仅含 no_tool_available 的列表
func TestBuildKiroTools_EmptyTools(t *testing.T) {
	result := buildKiroTools(nil)
	if len(result) != 1 {
		t.Fatalf("期望 1 个工具，实际 %d 个", len(result))
	}
	if result[0].ToolSpecification.Name != "no_tool_available" {
		t.Errorf("期望工具名为 no_tool_available，实际 %s", result[0].ToolSpecification.Name)
	}
}

// TestBuildKiroTools_FilterWebSearch 包含 web_search/websearch（大小写变体），应被过滤
func TestBuildKiroTools_FilterWebSearch(t *testing.T) {
	tools := []providers.Tool{
		makeTool("web_search", "搜索工具"),
		makeTool("WebSearch", "搜索工具2"),
		makeTool("WEBSEARCH", "搜索工具3"),
		makeTool("my_tool", "我的工具"),
	}
	result := buildKiroTools(tools)
	for _, t2 := range result {
		name := strings.ToLower(t2.ToolSpecification.Name)
		if name == "web_search" || name == "websearch" {
			t.Errorf("web_search/websearch 应被过滤，但仍存在: %s", t2.ToolSpecification.Name)
		}
	}
	// my_tool 应保留
	found := false
	for _, t2 := range result {
		if t2.ToolSpecification.Name == "my_tool" {
			found = true
			break
		}
	}
	if !found {
		t.Error("my_tool 应被保留")
	}
}

// TestBuildKiroTools_AllFilteredFallback 过滤后为空，应返回 no_tool_available
func TestBuildKiroTools_AllFilteredFallback(t *testing.T) {
	tools := []providers.Tool{
		makeTool("web_search", "搜索工具"),
		makeTool("websearch", "搜索工具2"),
	}
	result := buildKiroTools(tools)
	if len(result) != 1 {
		t.Fatalf("期望 1 个工具，实际 %d 个", len(result))
	}
	if result[0].ToolSpecification.Name != "no_tool_available" {
		t.Errorf("期望工具名为 no_tool_available，实际 %s", result[0].ToolSpecification.Name)
	}
}

// TestBuildKiroTools_EmptyDescription 工具描述为空或纯空白，应被过滤
func TestBuildKiroTools_EmptyDescription(t *testing.T) {
	tools := []providers.Tool{
		makeTool("tool_empty_desc", ""),
		makeTool("tool_whitespace_desc", "   "),
		makeTool("tool_valid", "有效描述"),
	}
	result := buildKiroTools(tools)
	for _, t2 := range result {
		if t2.ToolSpecification.Name == "tool_empty_desc" || t2.ToolSpecification.Name == "tool_whitespace_desc" {
			t.Errorf("空描述工具 %s 应被过滤", t2.ToolSpecification.Name)
		}
	}
	found := false
	for _, t2 := range result {
		if t2.ToolSpecification.Name == "tool_valid" {
			found = true
			break
		}
	}
	if !found {
		t.Error("有效工具 tool_valid 应被保留")
	}
}

// TestBuildKiroTools_TruncateDescription 描述长度超过 9216，应截断为 9216 字符并追加 "..."
func TestBuildKiroTools_TruncateDescription(t *testing.T) {
	longDesc := strings.Repeat("a", maxDescriptionLength+100)
	tools := []providers.Tool{makeTool("my_tool", longDesc)}
	result := buildKiroTools(tools)
	if len(result) == 0 {
		t.Fatal("期望返回工具列表非空")
	}
	desc := result[0].ToolSpecification.Description
	expectedLen := maxDescriptionLength + 3 // "..." 3 个字符
	if len(desc) != expectedLen {
		t.Errorf("期望描述长度 %d，实际 %d", expectedLen, len(desc))
	}
	if !strings.HasSuffix(desc, "...") {
		t.Error("期望描述以 ... 结尾")
	}
}

// TestBuildKiroTools_NormalDescription 描述长度未超过 9216，应保持原样
func TestBuildKiroTools_NormalDescription(t *testing.T) {
	desc := strings.Repeat("b", maxDescriptionLength-1)
	tools := []providers.Tool{makeTool("my_tool", desc)}
	result := buildKiroTools(tools)
	if len(result) == 0 {
		t.Fatal("期望返回工具列表非空")
	}
	if result[0].ToolSpecification.Description != desc {
		t.Error("描述未超过最大长度，不应被截断")
	}
}

// ============================================================
// 任务 9：Assistant 消息处理（buildAssistantMessage）与 Tool 消息测试
// ============================================================

// TestBuildAssistantMessage_WithThinkingAndContent 同时有 ReasoningContent 和文本，格式应为 <thinking>...</thinking>\n\ncontent
func TestBuildAssistantMessage_WithThinkingAndContent(t *testing.T) {
	msg := providers.Message{
		Role:             providers.RoleAssistant,
		Content:          "正文内容",
		ReasoningContent: "思考内容",
	}
	result := buildAssistantMessage(msg)
	expected := "<thinking>思考内容</thinking>\n\n正文内容"
	if result.Content != expected {
		t.Errorf("期望内容 %q，实际 %q", expected, result.Content)
	}
}

// TestBuildAssistantMessage_ThinkingOnly 只有 ReasoningContent，格式应为 <thinking>...</thinking>
func TestBuildAssistantMessage_ThinkingOnly(t *testing.T) {
	msg := providers.Message{
		Role:             providers.RoleAssistant,
		ReasoningContent: "思考内容",
	}
	result := buildAssistantMessage(msg)
	expected := "<thinking>思考内容</thinking>"
	if result.Content != expected {
		t.Errorf("期望内容 %q，实际 %q", expected, result.Content)
	}
}

// TestBuildAssistantMessage_ToolCalls_ValidJSON ToolCall.Function.Arguments 为有效 JSON，ToolUse.Input 应为解析后的 map
func TestBuildAssistantMessage_ToolCalls_ValidJSON(t *testing.T) {
	msg := providers.Message{
		Role: providers.RoleAssistant,
		ToolCalls: []providers.ToolCall{
			{
				Type: "function",
				ID:   "call-1",
				Function: providers.FunctionDefinitionParam{
					Name:      "my_func",
					Arguments: []byte(`{"key": "value", "num": 42}`),
				},
			},
		},
	}
	result := buildAssistantMessage(msg)
	if len(result.ToolUses) != 1 {
		t.Fatalf("期望 1 个 ToolUse，实际 %d 个", len(result.ToolUses))
	}
	tu := result.ToolUses[0]
	if tu.ToolUseId != "call-1" {
		t.Errorf("期望 ToolUseId=call-1，实际 %s", tu.ToolUseId)
	}
	if tu.Name != "my_func" {
		t.Errorf("期望 Name=my_func，实际 %s", tu.Name)
	}
	inputMap, ok := tu.Input.(map[string]any)
	if !ok {
		t.Fatalf("期望 Input 为 map[string]any，实际 %T", tu.Input)
	}
	if inputMap["key"] != "value" {
		t.Errorf("期望 key=value，实际 %v", inputMap["key"])
	}
}

// TestBuildAssistantMessage_ToolCalls_EmptyArguments Arguments 为空，ToolUse.Input 应为空 map {}
func TestBuildAssistantMessage_ToolCalls_EmptyArguments(t *testing.T) {
	msg := providers.Message{
		Role: providers.RoleAssistant,
		ToolCalls: []providers.ToolCall{
			{
				Type: "tool_use",
				ID:   "call-2",
				Function: providers.FunctionDefinitionParam{
					Name:      "empty_func",
					Arguments: nil,
				},
			},
		},
	}
	result := buildAssistantMessage(msg)
	if len(result.ToolUses) != 1 {
		t.Fatalf("期望 1 个 ToolUse，实际 %d 个", len(result.ToolUses))
	}
	inputMap, ok := result.ToolUses[0].Input.(map[string]any)
	if !ok {
		t.Fatalf("期望 Input 为 map[string]any，实际 %T", result.ToolUses[0].Input)
	}
	if len(inputMap) != 0 {
		t.Errorf("期望 Input 为空 map，实际 %v", inputMap)
	}
}

// TestBuildAssistantMessage_ToolCalls_InvalidType ToolCall.Type 不为 "function" 或 "tool_use"，该工具调用应被忽略
func TestBuildAssistantMessage_ToolCalls_InvalidType(t *testing.T) {
	msg := providers.Message{
		Role: providers.RoleAssistant,
		ToolCalls: []providers.ToolCall{
			{
				Type: "invalid_type",
				ID:   "call-3",
				Function: providers.FunctionDefinitionParam{
					Name:      "ignored_func",
					Arguments: []byte(`{}`),
				},
			},
		},
	}
	result := buildAssistantMessage(msg)
	if len(result.ToolUses) != 0 {
		t.Errorf("期望 ToolUses 为空（无效 type 应被忽略），实际 %d 个", len(result.ToolUses))
	}
}

// TestConvertRequest_RoleTool_AsLastMessage 最后一条为 RoleTool 消息，应转换为 ToolResult
func TestConvertRequest_RoleTool_AsLastMessage(t *testing.T) {
	req := newReq("claude-sonnet-4.5",
		makeUserMsg("用户消息"),
		makeAssistantMsg("助手回复"),
		makeToolMsg("tool-use-id-1", "工具执行结果"),
	)
	result, err := ConvertRequest(context.Background(), req)
	if err != nil {
		t.Fatalf("期望 error 为 nil，实际 %v", err)
	}
	if result == nil {
		t.Fatal("期望返回非 nil 结果")
	}
	ctx := result.ConversationState.CurrentMessage.UserInputMessage.UserInputMessageContext
	if ctx == nil {
		t.Fatal("期望 UserInputMessageContext 非 nil")
	}
	if len(ctx.ToolResults) == 0 {
		t.Fatal("期望 ToolResults 非空")
	}
	tr := ctx.ToolResults[0]
	if tr.ToolUseId != "tool-use-id-1" {
		t.Errorf("期望 ToolUseId=tool-use-id-1，实际 %s", tr.ToolUseId)
	}
	if tr.Status != "success" {
		t.Errorf("期望 Status=success，实际 %s", tr.Status)
	}
	if len(tr.Content) == 0 || tr.Content[0].Text != "工具执行结果" {
		t.Errorf("期望 ToolResult Content 为 %q，实际 %v", "工具执行结果", tr.Content)
	}
}

// TestDeduplicateToolResults 存在重复 toolUseId，应只保留首次出现的记录
func TestDeduplicateToolResults(t *testing.T) {
	toolResults := []ToolResult{
		{ToolUseId: "id-1", Status: "success", Content: []ToolResultContent{{Text: "first"}}},
		{ToolUseId: "id-2", Status: "success", Content: []ToolResultContent{{Text: "second"}}},
		{ToolUseId: "id-1", Status: "success", Content: []ToolResultContent{{Text: "duplicate"}}},
	}
	result := deduplicateToolResults(toolResults)
	if len(result) != 2 {
		t.Fatalf("期望去重后 2 条，实际 %d 条", len(result))
	}
	if result[0].ToolUseId != "id-1" || result[0].Content[0].Text != "first" {
		t.Error("期望保留首次出现的 id-1 记录")
	}
	if result[1].ToolUseId != "id-2" {
		t.Error("期望保留 id-2 记录")
	}
}

// ============================================================
// 任务 10：复杂多轮对话端到端测试
// ============================================================

// TestConvertRequest_FullConversation system + user + assistant(含 ToolCalls) + tool + user 完整对话
func TestConvertRequest_FullConversation(t *testing.T) {
	req := &providers.Request{
		Model: "claude-sonnet-4.5",
		Messages: []providers.Message{
			makeSystemMsg("你是一个助手"),
			makeUserMsg("请帮我查询天气"),
			{
				Role:    providers.RoleAssistant,
				Content: "我来帮你查询",
				ToolCalls: []providers.ToolCall{
					{
						Type: "function",
						ID:   "call-weather",
						Function: providers.FunctionDefinitionParam{
							Name:      "get_weather",
							Arguments: []byte(`{"city": "北京"}`),
						},
					},
				},
			},
			makeToolMsg("call-weather", "北京今天晴，25°C"),
			makeUserMsg("谢谢"),
		},
	}
	result, err := ConvertRequest(context.Background(), req)
	if err != nil {
		t.Fatalf("期望 error 为 nil，实际 %v", err)
	}
	if result == nil {
		t.Fatal("期望返回非 nil 结果")
	}

	history := result.ConversationState.History
	if len(history) == 0 {
		t.Fatal("期望 history 非空")
	}

	// 验证 history 中存在 assistant 消息且包含 ToolUses
	var foundAssistantWithToolUse bool
	for _, item := range history {
		if item.AssistantResponseMessage != nil && len(item.AssistantResponseMessage.ToolUses) > 0 {
			foundAssistantWithToolUse = true
			tu := item.AssistantResponseMessage.ToolUses[0]
			if tu.Name != "get_weather" {
				t.Errorf("期望工具名 get_weather，实际 %s", tu.Name)
			}
			if tu.ToolUseId != "call-weather" {
				t.Errorf("期望 ToolUseId=call-weather，实际 %s", tu.ToolUseId)
			}
		}
	}
	if !foundAssistantWithToolUse {
		t.Error("期望 history 中存在包含 ToolUses 的 assistant 消息")
	}

	// 验证 currentMessage 内容为 "谢谢"
	currentContent := result.ConversationState.CurrentMessage.UserInputMessage.Content
	if currentContent != "谢谢" {
		t.Errorf("期望 currentContent=%q，实际=%q", "谢谢", currentContent)
	}
}

// TestConvertRequest_AssistantWithToolCall_ThenToolResult assistant 带工具调用 + tool 结果作为最后消息
func TestConvertRequest_AssistantWithToolCall_ThenToolResult(t *testing.T) {
	req := &providers.Request{
		Model: "claude-sonnet-4.5",
		Messages: []providers.Message{
			makeUserMsg("请执行工具"),
			{
				Role: providers.RoleAssistant,
				ToolCalls: []providers.ToolCall{
					{
						Type: "function",
						ID:   "call-tool-1",
						Function: providers.FunctionDefinitionParam{
							Name:      "do_something",
							Arguments: []byte(`{"param": "value"}`),
						},
					},
				},
			},
			makeToolMsg("call-tool-1", "工具执行完成"),
		},
	}
	result, err := ConvertRequest(context.Background(), req)
	if err != nil {
		t.Fatalf("期望 error 为 nil，实际 %v", err)
	}
	if result == nil {
		t.Fatal("期望返回非 nil 结果")
	}

	// history 中 assistant 消息应包含 ToolUses
	history := result.ConversationState.History
	var foundAssistant bool
	for _, item := range history {
		if item.AssistantResponseMessage != nil {
			foundAssistant = true
			if len(item.AssistantResponseMessage.ToolUses) == 0 {
				t.Error("期望 history 中 assistant 消息包含 ToolUses")
			} else {
				if item.AssistantResponseMessage.ToolUses[0].ToolUseId != "call-tool-1" {
					t.Errorf("期望 ToolUseId=call-tool-1，实际 %s", item.AssistantResponseMessage.ToolUses[0].ToolUseId)
				}
			}
		}
	}
	if !foundAssistant {
		t.Error("期望 history 中存在 assistant 消息")
	}

	// currentMessage 应包含 ToolResults
	ctx := result.ConversationState.CurrentMessage.UserInputMessage.UserInputMessageContext
	if ctx == nil {
		t.Fatal("期望 UserInputMessageContext 非 nil")
	}
	if len(ctx.ToolResults) == 0 {
		t.Fatal("期望 ToolResults 非空")
	}
	if ctx.ToolResults[0].ToolUseId != "call-tool-1" {
		t.Errorf("期望 ToolResult.ToolUseId=call-tool-1，实际 %s", ctx.ToolResults[0].ToolUseId)
	}
}

// TestConvertRequest_ConsecutiveUserMessages_AutoContinue
// 验证：当 history 末尾为 user 类型消息（非 assistant）时，自动补充 AssistantResponseMessage{Content:"Continue"}
// 场景：user + assistant + user + user（最后两条 user 被合并为一条，history 末尾为 assistant，不触发）
// 改为：user + assistant + tool + user，tool 消息在 history 中作为 UserInputMessage，末尾为 user 类型，触发自动补充
func TestConvertRequest_ConsecutiveUserMessages_AutoContinue(t *testing.T) {
	req := newReq("claude-sonnet-4.5",
		makeUserMsg("第一条"),
		makeAssistantMsg("助手回复"),
		makeToolMsg("tool-id-x", "工具结果"),
		makeUserMsg("最后一条"),
	)
	result, err := ConvertRequest(context.Background(), req)
	if err != nil {
		t.Fatalf("期望 error 为 nil，实际 %v", err)
	}
	if result == nil {
		t.Fatal("期望返回非 nil 结果")
	}

	history := result.ConversationState.History
	if len(history) == 0 {
		t.Fatal("期望 history 非空")
	}

	// history 末尾应为 AssistantResponseMessage{Content:"Continue"}（tool 消息作为 UserInputMessage 在 history 末尾，触发自动补充）
	lastItem := history[len(history)-1]
	if lastItem.AssistantResponseMessage == nil {
		t.Errorf("期望 history 末尾为 AssistantResponseMessage，实际 %+v", lastItem)
	} else if lastItem.AssistantResponseMessage.Content != "Continue" {
		t.Errorf("期望自动补充的 assistant 内容为 Continue，实际 %q", lastItem.AssistantResponseMessage.Content)
	}
}
