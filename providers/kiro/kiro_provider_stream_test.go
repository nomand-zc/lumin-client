package kiro

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewProviderInit 验证 NewProvider 正确初始化所有必要字段
func TestNewProviderInit(t *testing.T) {
	provider := NewProvider("test-provider")

	assert.NotNil(t, provider)
	assert.Equal(t, "test-provider", provider.name)
	assert.NotNil(t, provider.options)
	assert.NotNil(t, provider.httpClient)
	assert.NotNil(t, provider.options.tokenConter, "tokenConter should be initialized via init()")
	assert.NotEmpty(t, provider.options.url, "url should be set to default")
	assert.NotEmpty(t, provider.options.headers, "headers should have default values")
}
