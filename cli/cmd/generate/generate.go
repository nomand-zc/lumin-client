package generate

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/nomand-zc/lumin-client/cli/internal/auth"
	"github.com/nomand-zc/lumin-client/cli/internal/factory"
	"github.com/nomand-zc/lumin-client/log"
	"github.com/nomand-zc/lumin-client/providers"
	"github.com/nomand-zc/lumin-client/utils"
)

// generator 持有 generate 命令的参数
type generator struct {
	credFile     string
	dataFile     string
	providerName string
	stream       bool
	provider     providers.Provider
}

var (
	defaultGenerator generator
)

// CMD 返回 generate 子命令
func CMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "生成 AI 内容",
		Long: `通过 AI Provider 生成内容，支持流式和非流式两种模式，支持自动刷新过期 token。

支持的 provider：
  - kiro

示例：
  # 非流式生成
  provider-client generate --creds /path/to/credentials.json --data /path/to/data.json --provider kiro
  # 流式生成
  provider-client generate --creds /path/to/credentials.json --data /path/to/data.json --provider kiro --stream`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return defaultGenerator.run()
		},
	}

	cmd.Flags().StringVarP(&defaultGenerator.credFile, "creds", "c", "", "凭证 JSON 文件路径或目录路径（必填）")
	cmd.Flags().StringVarP(&defaultGenerator.dataFile, "data", "d", "", "请求 JSON 文件路径（必填）")
	cmd.Flags().StringVarP(&defaultGenerator.providerName, "provider", "p", "kiro", fmt.Sprintf("provider 名称，支持：%v（必填）", factory.SupportedProviders))
	cmd.Flags().BoolVarP(&defaultGenerator.stream, "stream", "s", false, "是否使用流式模式（默认为非流式）")

	_ = cmd.MarkFlagRequired("creds")
	_ = cmd.MarkFlagRequired("data")

	return cmd
}

// run 执行生成逻辑
func (g *generator) run() error {
	// 初始化 provider
	var err error
	g.provider, err = factory.NewProvider(g.providerName)
	if err != nil {
		return err
	}

	creds, err := auth.GetValidCredentials(g.provider, g.credFile)
	if err != nil {
		return fmt.Errorf("凭证不可用: %w", err)
	}

	// 读取请求文件
	reqData, err := os.ReadFile(g.dataFile)
	if err != nil {
		return fmt.Errorf("读取请求文件失败: %w", err)
	}

	// 解析请求
	var req providers.Request
	if err := json.Unmarshal(reqData, &req); err != nil {
		return fmt.Errorf("解析请求 JSON 失败: %w", err)
	}

	// 将凭证设置到请求中
	req.Credential = creds

	log.Infof("\n ===== 开始生成 ===== \n原始请求: %s\n", utils.Bytes2Str(reqData))
	if g.stream {
		return g.runStream(&req)
	}
	return g.runNonStream(&req)
}

// runStream 执行流式生成逻辑
func (g *generator) runStream(req *providers.Request) error {
	// 确保启用流式模式
	req.GenerationConfig.Stream = true

	reader, err := g.provider.GenerateContentStream(context.Background(), req)
	if err != nil {
		return fmt.Errorf("流式生成失败: %w", err)
	}

	// 处理流式响应
	if err := reader.Each(context.Background(), func(resp *providers.Response) error {
		rspData, jsonErr := json.MarshalIndent(resp, "", "  ")
		if jsonErr != nil {
			log.Errorf("序列化响应失败: %v", jsonErr)
			return nil
		}
		log.Infof("\n[generator-runStream] Response Chunk: %s\n", utils.Bytes2Str(rspData))
		return nil
	}); err != nil {
		return fmt.Errorf("读取流式响应失败: %w", err)
	}

	log.Infof("\n ===== 流式生成完成 =====")
	return nil
}

// runNonStream 执行非流式生成逻辑
func (g *generator) runNonStream(req *providers.Request) error {
	req.GenerationConfig.Stream = false

	resp, err := g.provider.GenerateContent(context.Background(), req)
	if err != nil {
		return fmt.Errorf("生成失败: %w", err)
	}

	// 输出响应
	rspData, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化响应失败: %w", err)
	}
	log.Infof("\n[generator-runNonStream] Response: %s\n", utils.Bytes2Str(rspData))

	log.Infof("\n ===== 生成完成 =====")
	return nil
}
