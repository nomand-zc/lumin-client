package providers

import (
	"context"

	"github.com/nomand-zc/lumin-client/credentials"
	"github.com/nomand-zc/lumin-client/queue"
	"github.com/nomand-zc/lumin-client/usagerule"
)

// Provider is an interface for a provider. It includes methods for generating content and generating content in a stream. The GenerateContent method takes a context and a request (which contains credentials), and returns a response or an error. The GenerateContentStream method takes the same parameters but returns a ResponseChain for streaming responses.
type Model interface {
	// GenerateContent generates content.
	GenerateContent(ctx context.Context, req *Request) (*Response, error)
	// GenerateContentStream generates content in a stream.
	GenerateContentStream(ctx context.Context, req *Request) (queue.Consumer[*Response], error)
}

// CredentialManager 凭证管理器接口，负责凭证的刷新。
type CredentialManager interface {
	// Refresh 刷新凭证（如刷新过期的 Token），直接修改入参 creds 中的字段。
	Refresh(ctx context.Context, creds credentials.Credential) error
}

// UsageLimiter is an interface for limiting usage. It includes methods for listing models and getting usage rules and stats.
type UsageLimiter interface {
	// Models lists models.
	// 默认支持的模型列表
	Models(ctx context.Context) ([]string, error)

	// ListModels lists models.
	// 获取当前凭证支持的模型列表
	ListModels(ctx context.Context, creds credentials.Credential) ([]string, error)

	// DefaultUsageRules 获取供应商默认的用量规则列表
	DefaultUsageRules(ctx context.Context) ([]*usagerule.UsageRule, error)

	// GetUsageRules 获取用量规则列表
	GetUsageRules(ctx context.Context, creds credentials.Credential) ([]*usagerule.UsageRule, error)

	// GetUsageStats 获取规则的用量统计信息
	GetUsageStats(ctx context.Context, creds credentials.Credential) ([]*usagerule.UsageStats, error)
}

// Provider is an interface for a provider. It includes methods for generating content, generating content in a stream, refreshing tokens, and limiting usage.
type Provider interface {
	Type() string
	Name() string
	Model
	CredentialManager
	UsageLimiter
}