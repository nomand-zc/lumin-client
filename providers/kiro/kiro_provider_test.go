package kiro

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewProviderOptionsIsolation 测试多个 Provider 实例的 options 是否完全隔离，不存在全局状态污染
func TestNewProviderOptionsIsolation(t *testing.T) {
	// RED: 验证当前代码中的 bug —— 多实例会共享全局 defaultOptions

	// 创建第一个 provider，不传自定义 Header
	provider1 := NewProvider("provider1")

	// 创建第二个 provider，添加自定义 Header
	provider2 := NewProvider("provider2", WithHeader("X-Custom-Header", "value2"))

	// 检查 provider1 的 headers —— 应该不含 X-Custom-Header
	// 如果存在全局污染，provider1 会被污染成包含 X-Custom-Header
	headers1 := provider1.options.headers
	_, hasCustomHeader1 := headers1["X-Custom-Header"]
	assert.False(t, hasCustomHeader1, "provider1 should not be affected by provider2's options")

	// 检查 provider2 的 headers —— 应该包含 X-Custom-Header
	headers2 := provider2.options.headers
	_, hasCustomHeader2 := headers2["X-Custom-Header"]
	assert.True(t, hasCustomHeader2, "provider2 should have X-Custom-Header")

	// 创建第三个 provider，用不同的 Header
	provider3 := NewProvider("provider3", WithHeader("X-Another-Header", "value3"))

	// 重新检查 provider1 和 provider2，确保它们不被 provider3 污染
	_, hasAnotherHeader1 := provider1.options.headers["X-Another-Header"]
	assert.False(t, hasAnotherHeader1, "provider1 should not be affected by provider3's options")

	_, hasAnotherHeader2 := provider2.options.headers["X-Another-Header"]
	assert.False(t, hasAnotherHeader2, "provider2 should not be affected by provider3's options")

	// 检查 provider3
	headers3 := provider3.options.headers
	_, hasAnotherHeader3 := headers3["X-Another-Header"]
	assert.True(t, hasAnotherHeader3, "provider3 should have X-Another-Header")
	_, hasCustomHeader3 := headers3["X-Custom-Header"]
	assert.False(t, hasCustomHeader3, "provider3 should not have X-Custom-Header from provider2")
}

// TestNewProviderMultipleOptionsApplication 测试多个 Option 的应用是否独立
func TestNewProviderMultipleOptionsApplication(t *testing.T) {
	// RED: 验证多个选项链式调用不会产生全局污染

	provider1 := NewProvider("provider1",
		WithHeader("Header1", "value1"),
		WithHeader("Header2", "value2"),
	)

	provider2 := NewProvider("provider2",
		WithHeader("HeaderA", "valueA"),
		WithHeader("HeaderB", "valueB"),
	)

	// 验证 provider1 不包含 provider2 的 headers
	assert.NotContains(t, provider1.options.headers, "HeaderA")
	assert.NotContains(t, provider1.options.headers, "HeaderB")
	assert.Contains(t, provider1.options.headers, "Header1")
	assert.Contains(t, provider1.options.headers, "Header2")

	// 验证 provider2 不包含 provider1 的 headers
	assert.NotContains(t, provider2.options.headers, "Header1")
	assert.NotContains(t, provider2.options.headers, "Header2")
	assert.Contains(t, provider2.options.headers, "HeaderA")
	assert.Contains(t, provider2.options.headers, "HeaderB")
}
