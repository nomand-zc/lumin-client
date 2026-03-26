package builder

import (
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/nomand-zc/lumin-client/providers"
	"github.com/nomand-zc/lumin-client/providers/kiro/converter/builder/types"
	"github.com/nomand-zc/lumin-client/utils"
)

// GetMessageText 获取消息的纯文本内容
func GetMessageText(msg providers.Message) string {
	if msg.Content != "" {
		return msg.Content
	}
	var parts []string
	for _, part := range msg.ContentParts {
		if part.Type == providers.ContentTypeText && part.Text != nil {
			parts = append(parts, *part.Text)
		}
	}
	return strings.Join(parts, "")
}

// ConvertImage 将 providers.Image 转换为 types.Image
func ConvertImage(img *providers.Image) *types.Image {
	if img == nil {
		return nil
	}

	format := img.Format
	if format == "" {
		format = "jpeg"
	}

	var bytesStr string
	if len(img.Data) > 0 {
		bytesStr = base64.StdEncoding.EncodeToString(img.Data)
	} else if img.URL != "" {
		// URL 格式的图片暂不支持直接转换，跳过
		return nil
	}

	if bytesStr == "" {
		return nil
	}

	return &types.Image{
		Format: format,
		Source: types.ImageSource{
			Bytes: bytesStr,
		},
	}
}

// ConvertSchema 将 providers.Schema 转换为 map[string]any
func ConvertSchema(schema *providers.Schema) any {
	if schema == nil {
		return map[string]any{"type": "object", "properties": map[string]any{}}
	}

	result := map[string]any{}
	if schema.Type != "" {
		result["type"] = schema.Type
	}
	if schema.Description != "" {
		result["description"] = schema.Description
	}
	if len(schema.Required) > 0 {
		result["required"] = schema.Required
	}
	if len(schema.Properties) > 0 {
		props := map[string]any{}
		for k, v := range schema.Properties {
			props[k] = ConvertSchema(v)
		}
		result["properties"] = props
	}
	return result
}

// ParseJSONOrString 尝试将字节解析为 JSON 对象，失败则返回原始字符串
func ParseJSONOrString(data []byte) any {
	if len(data) == 0 {
		return map[string]any{}
	}
	trimmed := strings.TrimSpace(utils.Bytes2Str(data))
	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		var result any
		if err := json.Unmarshal(data, &result); err == nil {
			return result
		}
	}
	return utils.Bytes2Str(data)
}

// DeduplicateToolResults 按 toolUseId 去重（保留首次出现）
func DeduplicateToolResults(toolResults []types.ToolResult) []types.ToolResult {
	seen := make(map[string]bool)
	var unique []types.ToolResult
	for _, tr := range toolResults {
		if !seen[tr.ToolUseId] {
			seen[tr.ToolUseId] = true
			unique = append(unique, tr)
		}
	}
	return unique
}
