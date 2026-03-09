package credentials

// CredentialStatus 凭证可用性状态
type CredentialStatus int

const (
	// StatusAvailable 当前可用，凭证有效且可正常使用
	StatusAvailable CredentialStatus = 1
	// StatusExpired Token 已过期，可通过刷新（Refresh）恢复
	StatusExpired CredentialStatus = 2
	// StatusInvalidated 永久失效，无法恢复（如 refresh token 无效，对应 ErrInvalidGrant）
	StatusInvalidated = 3
	// StatusBanned 被平台封禁（如 Kiro 账号被 AWS 临时封禁，body 含 TEMPORARILY_SUSPENDED）
	StatusBanned CredentialStatus = 4
	// StatusUsageLimited 触发了 usageRule 的用量限制
	StatusUsageLimited CredentialStatus = 5
	// StatusReauthRequired 需要用户重新走授权流程（如 GeminiCLI 的 VALIDATION_REQUIRED）
	StatusReauthRequired CredentialStatus = 6
)

// IsAvailable 判断凭证是否可用
func (s CredentialStatus) IsAvailable() bool {
	return s == StatusAvailable
}

// String 返回 CredentialStatus 的可读字符串表示
func (s CredentialStatus) String() string {
	switch s {
	case StatusAvailable:
		return "available"
	case StatusExpired:
		return "expired"
	case StatusInvalidated:
		return "invalidated"
	case StatusBanned:
		return "banned"
	case StatusUsageLimited:
		return "usage_limited"
	case StatusReauthRequired:
		return "reauth_required"
	default:
		return "unknown"
	}
}
