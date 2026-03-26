package credentials

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestStatusInvalidatedType 验证 StatusInvalidated 是 CredentialStatus 类型而非 untyped int
func TestStatusInvalidatedType(t *testing.T) {
	// 验证 StatusInvalidated 可以赋值给 CredentialStatus 变量（类型正确则编译通过）
	var s CredentialStatus = StatusInvalidated
	assert.Equal(t, CredentialStatus(3), s, "StatusInvalidated should equal CredentialStatus(3)")
}

// TestCredentialStatusConstants 验证所有 CredentialStatus 常量的值和类型一致性
func TestCredentialStatusConstants(t *testing.T) {
	tests := []struct {
		name   string
		status CredentialStatus
		value  CredentialStatus
	}{
		{"StatusAvailable", StatusAvailable, 1},
		{"StatusExpired", StatusExpired, 2},
		{"StatusInvalidated", StatusInvalidated, 3},
		{"StatusBanned", StatusBanned, 4},
		{"StatusUsageLimited", StatusUsageLimited, 5},
		{"StatusReauthRequired", StatusReauthRequired, 6},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.value, tt.status, "%s should equal %d", tt.name, tt.value)
		})
	}
}

// TestCredentialStatusString 验证 String() 方法返回正确的描述
func TestCredentialStatusString(t *testing.T) {
	tests := []struct {
		status CredentialStatus
		want   string
	}{
		{StatusAvailable, "available"},
		{StatusExpired, "expired"},
		{StatusInvalidated, "invalidated"},
		{StatusBanned, "banned"},
		{StatusUsageLimited, "usage_limited"},
		{StatusReauthRequired, "reauth_required"},
		{CredentialStatus(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.status.String())
		})
	}
}

// TestCredentialStatusIsAvailable 验证 IsAvailable() 方法只对 StatusAvailable 返回 true
func TestCredentialStatusIsAvailable(t *testing.T) {
	assert.True(t, StatusAvailable.IsAvailable())
	assert.False(t, StatusExpired.IsAvailable())
	assert.False(t, StatusInvalidated.IsAvailable())
	assert.False(t, StatusBanned.IsAvailable())
	assert.False(t, StatusUsageLimited.IsAvailable())
	assert.False(t, StatusReauthRequired.IsAvailable())
}
