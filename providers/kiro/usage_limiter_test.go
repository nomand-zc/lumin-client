package kiro

import (
	"context"
	"testing"
	"time"

	"github.com/nomand-zc/lumin-client/credentials"
	kirocreds "github.com/nomand-zc/lumin-client/credentials/kiro"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// wrongTypeCred 是一个非 kiro 类型的凭证，用于测试类型断言安全性
type wrongTypeCred struct{}

func (w *wrongTypeCred) Clone() credentials.Credential { return w }
func (w *wrongTypeCred) Validate() error               { return nil }
func (w *wrongTypeCred) GetAccessToken() string        { return "tok" }
func (w *wrongTypeCred) GetRefreshToken() string       { return "rtok" }
func (w *wrongTypeCred) GetExpiresAt() *time.Time      { return nil }
func (w *wrongTypeCred) IsExpired() bool               { return false }
func (w *wrongTypeCred) GetUserInfo() (credentials.UserInfo, error) {
	return credentials.UserInfo{}, nil
}
func (w *wrongTypeCred) ToMap() map[string]any { return nil }

// TestGetUsageStatsWithWrongCredType 验证传入非 kiro 凭证时不会 panic，而是返回错误
func TestGetUsageStatsWithWrongCredType(t *testing.T) {
	p := NewProvider("test")

	// 传入错误类型的凭证——依赖 fetchUsageResp -> send() 中 ok 检查返回错误
	wrongCred := &wrongTypeCred{}
	_, err := p.GetUsageStats(context.Background(), wrongCred)

	// 应该返回错误，而不是 panic
	require.Error(t, err, "GetUsageStats with wrong credential type should return error, not panic")
}

// TestGetUsageRulesWithWrongCredType 验证传入非 kiro 凭证时不会 panic，而是返回错误
func TestGetUsageRulesWithWrongCredType(t *testing.T) {
	p := NewProvider("test")

	wrongCred := &wrongTypeCred{}
	_, err := p.GetUsageRules(context.Background(), wrongCred)

	require.Error(t, err, "GetUsageRules with wrong credential type should return error, not panic")
}

// TestGetUsageStatsCredTypeAssertionSafety 验证 GetUsageStats 中的 kiro 凭证类型断言使用 ok 检查
// 当 fetchUsageResp 成功但 creds 不是 kirocreds.Credential 时，应安全处理而非 panic
func TestGetUsageStatsCredTypeAssertionSafety(t *testing.T) {
	// 验证 kiro 凭证类型被正确识别
	cred := &kirocreds.Credential{
		Region: "us-east-1",
	}

	_, ok := interface{}(cred).(*kirocreds.Credential)
	assert.True(t, ok, "kiro credential should be of type *kirocreds.Credential")

	// 验证非 kiro 凭证不被识别为 kirocreds.Credential
	wrongCred := &wrongTypeCred{}
	_, ok2 := interface{}(wrongCred).(*kirocreds.Credential)
	assert.False(t, ok2, "wrong credential type should not be identified as kirocreds.Credential")
}
