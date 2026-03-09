package providers

import (
	"context"

	"github.com/nomand-zc/lumin/credentials"
	"github.com/nomand-zc/lumin/queue"
	"github.com/nomand-zc/lumin/usagerule"
)

// Provider is an interface for a provider. It includes methods for generating content and generating content in a stream. The GenerateContent method takes a context, credentials, and a request, and returns a response or an error. The GenerateContentStream method takes the same parameters but returns a ResponseChain for streaming responses.
type Model interface {
	// GenerateContent generates content.
	GenerateContent(ctx context.Context, creds credentials.Credential, req Request) (*Response, error)
	// GenerateContentStream generates content in a stream.
	GenerateContentStream(ctx context.Context, creds credentials.Credential, req Request) (queue.Consumer[*Response], error)
}

// CredentialManager 凭证管理器接口，负责凭证的刷新和可用性校验。
type CredentialManager interface {
	// Refresh 刷新凭证（如刷新过期的 Token），直接修改入参 creds 中的字段。
	Refresh(ctx context.Context, creds credentials.Credential) error
	// CheckAvailability 校验凭证的可用性，返回凭证当前的状态。
	CheckAvailability(ctx context.Context, creds credentials.Credential) (credentials.CredentialStatus, error)
}

// UsageLimiter is an interface for limiting usage. It includes methods for listing models and getting usage rules and stats.
type UsageLimiter interface {
	// Models lists models.
	// 默认支持的模型列表
	Models(ctx context.Context) ([]string, error)

	// ListModels lists models.
	// 获取当前凭证支持的模型列表
	ListModels(ctx context.Context, creds credentials.Credential) ([]string, error)

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