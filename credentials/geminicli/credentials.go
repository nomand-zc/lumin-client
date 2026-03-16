package geminicli

import (
	"encoding/json"
	"time"

	"github.com/nomand-zc/lumin-client/credentials"
)

// Google OAuth2 固定常量
const (
	DefaultClientID     = "681255809395-oo8ft2oprdrnp9e3aqf6av3hmdib135j.apps.googleusercontent.com"
	DefaultClientSecret = "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"
	DefaultTokenURI     = "https://oauth2.googleapis.com/token"
)

func init() {
	credentials.Register("geminicli", NewCredential[[]byte])
}

// Credential GeminiCLI 凭证结构体
type Credential struct {
	// AccessToken 当前访问令牌
	AccessToken string `json:"access_token"`
	// RefreshToken 刷新令牌
	RefreshToken string `json:"refresh_token,omitempty"`
	// Token 原始 token 值（与 access_token 相同，保持与凭证 JSON 兼容）
	Token string `json:"token,omitempty"`
	// ClientID OAuth2 客户端 ID
	ClientID string `json:"client_id,omitempty"`
	// ClientSecret OAuth2 客户端密钥
	ClientSecret string `json:"client_secret,omitempty"`
	// ProjectID Google Cloud 项目 ID
	ProjectID string `json:"project_id,omitempty"`
	// Email 用户邮箱
	Email string `json:"email,omitempty"`
	// Scopes OAuth2 权限范围
	Scopes []string `json:"scopes,omitempty"`
	// TokenURI Token 端点 URL
	TokenURI string `json:"token_uri,omitempty"`
	// ExpiresAt 过期时间（对应 JSON 中的 expiry 字段）
	ExpiresAt *time.Time `json:"expiry,omitempty"`

	raw map[string]any `json:"-"` // 原始凭证数据
}

// NewCredential 创建一个新的凭据实例
// 支持传入 JSON 字符串或 []byte，解析失败时返回 nil
func NewCredential[T string | []byte](raw T) credentials.Credential {
	var creds Credential
	if err := json.Unmarshal([]byte(raw), &creds); err != nil {
		return nil
	}

	// 兼容处理：如果 access_token 为空但 token 不为空，则用 token 填充
	if creds.AccessToken == "" && creds.Token != "" {
		creds.AccessToken = creds.Token
	}
	// 反向同步：如果 token 为空但 access_token 不为空
	if creds.Token == "" && creds.AccessToken != "" {
		creds.Token = creds.AccessToken
	}

	// 填充默认值
	if creds.ClientID == "" {
		creds.ClientID = DefaultClientID
	}
	if creds.ClientSecret == "" {
		creds.ClientSecret = DefaultClientSecret
	}
	if creds.TokenURI == "" {
		creds.TokenURI = DefaultTokenURI
	}

	return &creds
}

// Clone 克隆凭据实例
func (c *Credential) Clone() credentials.Credential {
	clone := *c
	if c.ExpiresAt != nil {
		t := *c.ExpiresAt
		clone.ExpiresAt = &t
	}
	if c.Scopes != nil {
		clone.Scopes = make([]string, len(c.Scopes))
		copy(clone.Scopes, c.Scopes)
	}
	clone.raw = nil
	return &clone
}

// Validate 校验凭据的格式有效性（仅校验格式，不校验是否过期）
func (c *Credential) Validate() error {
	if c == nil {
		return credentials.ErrCredentialEmpty
	}
	if c.AccessToken == "" {
		return credentials.ErrAccessTokenEmpty
	}
	if c.RefreshToken == "" {
		return credentials.ErrRefreshTokenEmpty
	}
	if c.ProjectID == "" {
		return credentials.ErrProjectIDEmpty
	}
	return nil
}

// GetAccessToken 返回访问令牌
func (c *Credential) GetAccessToken() string {
	return c.AccessToken
}

// GetRefreshToken 返回刷新令牌
func (c *Credential) GetRefreshToken() string {
	return c.RefreshToken
}

// GetExpiresAt 返回过期时间
func (c *Credential) GetExpiresAt() *time.Time {
	return c.ExpiresAt
}

// GetUserInfo 返回用户信息
func (c *Credential) GetUserInfo() (credentials.UserInfo, error) {
	if c == nil {
		return credentials.UserInfo{}, nil
	}
	return credentials.UserInfo{
		Email: c.Email,
	}, nil
}

// IsExpired 检查凭据是否过期
func (c *Credential) IsExpired() bool {
	if c.ExpiresAt == nil {
		return true
	}
	return time.Now().After(*c.ExpiresAt)
}

// ToMap 将凭据转换为 map 格式
func (c *Credential) ToMap() map[string]any {
	if c == nil {
		return nil
	}
	if c.raw == nil {
		c.raw = map[string]any{
			"access_token":  c.AccessToken,
			"refresh_token": c.RefreshToken,
			"token":         c.Token,
			"client_id":     c.ClientID,
			"client_secret": c.ClientSecret,
			"project_id":    c.ProjectID,
			"email":         c.Email,
			"scopes":        c.Scopes,
			"token_uri":     c.TokenURI,
			"expiry":        c.ExpiresAt,
		}
	}
	return c.raw
}

// ResetRaw 重置 raw 缓存，使 ToMap() 下次调用时重新构建
func (c *Credential) ResetRaw() {
	c.raw = nil
}
