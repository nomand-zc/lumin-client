package geminicli

import (
	"fmt"
	"maps"
	"runtime"
)

const (
	// Gemini CLI API 端点
	DefaultEndpoint = "https://cloudcode-pa.googleapis.com"
	// API 版本
	DefaultAPIVersion = "v1internal"

	// GeminiCLI 版本号，对齐 gemini-cli package.json 中的 version
	GeminiCLIVersion = "0.36.0"
	// X-Goog-Api-Client header 值，对齐 @google/genai SDK 版本
	GeminiCLIApiClientHeader = "google-genai-sdk/1.30.0 gl-node/v22.19.0"
)

var defaultOptions = Options{
	endpoint:   DefaultEndpoint,
	apiVersion: DefaultAPIVersion,
	headers: map[string]string{
		"Content-Type":     "application/json",
		"X-Goog-Api-Client": GeminiCLIApiClientHeader,
	},
}

// Options 配置选项
type Options struct {
	endpoint   string
	apiVersion string
	headers    map[string]string
}

// Option 配置选项函数
type Option func(*Options)

// WithEndpoint 设置 API 端点
func WithEndpoint(endpoint string) Option {
	return func(o *Options) {
		o.endpoint = endpoint
	}
}

// WithAPIVersion 设置 API 版本
func WithAPIVersion(version string) Option {
	return func(o *Options) {
		o.apiVersion = version
	}
}

// WithHeader 设置单个 Header
func WithHeader(key, value string) Option {
	return func(o *Options) {
		if o.headers == nil {
			o.headers = make(map[string]string)
		}
		o.headers[key] = value
	}
}

// WithHeaders 合并多个 Header
func WithHeaders(headers map[string]string) Option {
	return func(o *Options) {
		if o.headers == nil {
			o.headers = make(map[string]string)
		}
		maps.Copy(o.headers, headers)
	}
}

// StreamURL 返回流式请求的完整 URL
func (o *Options) StreamURL() string {
	return o.endpoint + "/" + o.apiVersion + ":streamGenerateContent?alt=sse"
}

// GenerateURL 返回非流式请求的完整 URL
func (o *Options) GenerateURL() string {
	return o.endpoint + "/" + o.apiVersion + ":generateContent"
}

// GeminiCLIUserAgent 生成符合 Gemini CLI 格式的 User-Agent 字符串
// 对齐 gemini-cli contentGenerator.ts 中的 User-Agent 生成逻辑
// 格式: "GeminiCLI/<version>/<model> (<os>; <arch>; <surface>)"
func GeminiCLIUserAgent(model string) string {
	if model == "" {
		model = "unknown"
	}
	return fmt.Sprintf("GeminiCLI/%s/%s (%s; %s; cli)", GeminiCLIVersion, model, geminiCLIOS(), geminiCLIArch())
}

// geminiCLIOS 将 Go runtime OS 名映射为 Node.js 风格的平台字符串
func geminiCLIOS() string {
	switch runtime.GOOS {
	case "windows":
		return "win32"
	default:
		return runtime.GOOS
	}
}

// geminiCLIArch 将 Go runtime 架构名映射为 Node.js 风格的架构字符串
func geminiCLIArch() string {
	switch runtime.GOARCH {
	case "amd64":
		return "x64"
	case "386":
		return "x86"
	default:
		return runtime.GOARCH
	}
}
