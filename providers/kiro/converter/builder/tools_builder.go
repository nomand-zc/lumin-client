package builder

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/nomand-zc/lumin-client/providers"
	"github.com/nomand-zc/lumin-client/providers/kiro/converter/builder/types"
)

const (
	// maxDescriptionLength 工具描述最大长度（Kiro API 限制 10240，留余量给 "..."）
	maxDescriptionLength = 10237
	// maxToolNameLength 工具名称最大长度（Kiro API 限制 64 字符）
	maxToolNameLength = 64
)

// placeholderTool 是当无可用工具时使用的占位工具
var placeholderTool = types.Tool{
	ToolSpecification: types.ToolSpecification{
		Name:        "no_tool_available",
		Description: "This is a placeholder tool when no other tools are available. It does nothing.",
		InputSchema: types.InputSchema{Json: map[string]any{"type": "object", "properties": map[string]any{}}},
	},
}

// ToolsBuilder 负责将 providers.Tool 列表转换为 types.Tool 列表：
//   - 过滤 web_search/websearch、空名称、空描述的工具
//   - 截断超长描述
//   - 若过滤后列表为空，使用占位工具（no_tool_available）填充
//
// 结果写入 BuildContext.KiroTools
type ToolsBuilder struct{}

// Build 实现 MessageBuilder 接口
func (b *ToolsBuilder) Build(ctx *BuildContext) error {
	ctx.KiroTools = buildKiroTools(ctx.Req.Tools)
	return nil
}

// buildKiroTools 将 providers.Tool 列表转换为 types.Tool 列表
func buildKiroTools(tools []providers.Tool) []types.Tool {
	if len(tools) == 0 {
		return []types.Tool{placeholderTool}
	}

	// 过滤 web_search / websearch 及空名称
	filtered := make([]providers.Tool, 0, len(tools))
	for _, tool := range tools {
		if tool.Name == "" {
			continue
		}
		name := strings.ToLower(tool.Name)
		if name == "web_search" || name == "websearch" {
			continue
		}
		filtered = append(filtered, tool)
	}

	if len(filtered) == 0 {
		return []types.Tool{placeholderTool}
	}

	// 截断工具名称 + 空描述兜底 + 截断超长描述
	kiroTools := make([]types.Tool, 0, len(filtered))
	for _, tool := range filtered {
		name := shortenToolNameIfNeeded(tool.Name)

		// 空描述时自动填充默认值（Kiro API 要求非空描述）
		desc := tool.Description
		if strings.TrimSpace(desc) == "" {
			desc = fmt.Sprintf("Tool: %s", name)
		}
		if len(desc) > maxDescriptionLength {
			// 安全截断：确保不截断 UTF-8 多字节字符
			truncLen := maxDescriptionLength - 30
			for truncLen > 0 && !utf8.RuneStart(desc[truncLen]) {
				truncLen--
			}
			desc = desc[:truncLen] + "... (description truncated)"
		}
		kiroTools = append(kiroTools, types.Tool{
			ToolSpecification: types.ToolSpecification{
				Name:        name,
				Description: desc,
				InputSchema: types.InputSchema{Json: ConvertSchema(&tool.Parameters)},
			},
		})
	}

	if len(kiroTools) == 0 {
		return []types.Tool{placeholderTool}
	}

	return kiroTools
}

// shortenToolNameIfNeeded 截断超过 64 字符的工具名称。
// MCP 工具通常有较长的名称，如 "mcp__server-name__tool-name"。
// 尽量保留 "mcp__" 前缀和最后一个分段。
func shortenToolNameIfNeeded(name string) string {
	if len(name) <= maxToolNameLength {
		return name
	}
	// 对于 MCP 工具，尝试保留前缀和最后一段
	if strings.HasPrefix(name, "mcp__") {
		idx := strings.LastIndex(name, "__")
		if idx > 0 {
			cand := "mcp__" + name[idx+2:]
			if len(cand) > maxToolNameLength {
				return cand[:maxToolNameLength]
			}
			return cand
		}
	}
	return name[:maxToolNameLength]
}
