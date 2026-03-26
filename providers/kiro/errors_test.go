package kiro

import (
	"testing"

	"github.com/nomand-zc/lumin-client/providers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestClassifyHTTPStatus 验证 HTTP 状态码归一化为标准 HTTPError 类型
func TestClassifyHTTPStatus(t *testing.T) {
	tests := []struct {
		name          string
		statusCode    int
		wantErrorType providers.ErrorType
		wantErrorCode int
	}{
		{
			name:          "401 未授权应归类为 Unauthorized",
			statusCode:    401,
			wantErrorType: providers.ErrorTypeUnauthorized,
			wantErrorCode: providers.ErrorCodeUnauthorized,
		},
		{
			name:          "403 禁止应归类为 Forbidden",
			statusCode:    403,
			wantErrorType: providers.ErrorTypeForbidden,
			wantErrorCode: providers.ErrorCodeForbidden,
		},
		{
			name:          "429 限流应归类为 RateLimit",
			statusCode:    429,
			wantErrorType: providers.ErrorTypeRateLimit,
			wantErrorCode: providers.ErrorCodeRateLimit,
		},
		{
			name:          "400 请求错误应归类为 BadRequest",
			statusCode:    400,
			wantErrorType: providers.ErrorTypeBadRequest,
			wantErrorCode: providers.ErrorCodeBadRequest,
		},
		{
			name:          "500 服务器错误应归类为 ServerError",
			statusCode:    500,
			wantErrorType: providers.ErrorTypeServerError,
			wantErrorCode: providers.ErrorCodeServerError,
		},
		{
			name:          "502 Bad Gateway 应归类为 ServerError",
			statusCode:    502,
			wantErrorType: providers.ErrorTypeServerError,
			wantErrorCode: providers.ErrorCodeServerError,
		},
		{
			name:          "503 服务不可用应归类为 ServerError",
			statusCode:    503,
			wantErrorType: providers.ErrorTypeServerError,
			wantErrorCode: providers.ErrorCodeServerError,
		},
		{
			name:          "402 Kiro月度配额耗尽应归类为 RateLimit",
			statusCode:    402,
			wantErrorType: providers.ErrorTypeRateLimit,
			wantErrorCode: providers.ErrorCodeRateLimit,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errType, errCode := classifyHTTPStatus(tt.statusCode)
			assert.Equal(t, tt.wantErrorType, errType)
			assert.Equal(t, tt.wantErrorCode, errCode)
		})
	}
}

// TestNewHTTPError 验证 newHTTPError 创建的 HTTPError 字段正确
func TestNewHTTPError(t *testing.T) {
	body := []byte(`{"error": "unauthorized"}`)
	err := newHTTPError(401, "unauthorized", body)

	require.NotNil(t, err)
	assert.Equal(t, providers.ErrorTypeUnauthorized, err.ErrorType)
	assert.Equal(t, providers.ErrorCodeUnauthorized, err.ErrorCode)
	assert.Equal(t, "unauthorized", err.Message)
	assert.Equal(t, 401, err.RawStatusCode)
	assert.Equal(t, body, err.RawBody)
}
