package converter

import (
	"context"
	"fmt"

	"github.com/nomand-zc/lumin-client/providers"
	"github.com/nomand-zc/lumin-client/providers/kiro/converter/builder"
)

// ConvertRequest 将通用请求转换为 Kiro CodeWhisperer 请求格式
// 采用流水线模式：按序调用各构建器，通过 BuildContext 传递中间状态
// 任意阶段标记 Done=true 时返回 (nil, nil)，构建失败时返回 (nil, error)
func ConvertRequest(ctx context.Context, req *providers.Request) (*KiroRequest, error) {
	if len(req.Messages) == 0 {
		return nil, fmt.Errorf("request messages is empty")
	}

	bCtx := &builder.BuildContext{
		Ctx:     ctx,
		Req:     req,
		ModelId: req.Model,
	}

	// 流水线：各阶段按序执行
	pipeline := []builder.MessageBuilder{
		&builder.PreprocessBuilder{},
		&builder.SystemPromptBuilder{},
		&builder.ToolsBuilder{},
		&builder.HistoryBuilder{},
		&builder.CurrentMessageBuilder{},
	}

	for _, b := range pipeline {
		if err := b.Build(bCtx); err != nil {
			return nil, fmt.Errorf("build message failed: %w", err)
		}
		if bCtx.Done {
			return nil, nil
		}
	}

	// 组装最终请求
	return builder.Assemble(bCtx), nil
}
