package credentials

import (
	"time"

	"github.com/nomand-zc/lumin-client/utils"
)

// Credential 凭证接口，用于向 Provider 进行身份认证。包含刷新凭证、获取 access/refresh token、检查过期、转换为 map 格式等方法。
type Credential interface {
	Clone() Credential
	// Validate validates the credentials.
	Validate() error
	// GetAccessToken returns the access token.
	GetAccessToken() string
	// GetRefreshToken returns the refresh token.
	GetRefreshToken() string
	// GetExpiresAt returns the expiration time of the credentials.
	GetExpiresAt() *time.Time
	// IsExpired returns true if the credentials are expired.
	IsExpired() bool
	// GetUserInfo returns the user info.
	GetUserInfo() (UserInfo, error)
	// ToMap converts the credentials to a map format for storage or transmission.
	ToMap() map[string]any
}

// UserInfo 用户信息
type UserInfo struct {
	ID    string `json:"id,omitempty"`
	Email string `json:"email,omitempty"`
}

// GetValue gets the value of the given key from the credentials. It returns the value and a boolean indicating whether the key exists and is of the correct type.
func GetValue[V any](creds Credential, key string) (V, bool) {
	return utils.GetMapValue[string, V](creds.ToMap(), key)
}
