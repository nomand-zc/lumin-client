package kiro

import (
	"fmt"

	"github.com/nomand-zc/lumin-client/providers"
)

// classifyHTTPStatus 根据 HTTP 状态码分类错误类型和错误码
// Kiro 特殊映射：402（月度配额耗尽）→ rate_limit，与 429 同等处理
func classifyHTTPStatus(code int) (providers.ErrorType, int) {
	switch code {
	case 402, 429:
		return providers.ErrorTypeRateLimit, providers.ErrorCodeRateLimit
	case 401:
		return providers.ErrorTypeUnauthorized, providers.ErrorCodeUnauthorized
	case 403:
		return providers.ErrorTypeForbidden, providers.ErrorCodeForbidden
	case 400:
		return providers.ErrorTypeBadRequest, providers.ErrorCodeBadRequest
	default:
		return providers.ErrorTypeServerError, providers.ErrorCodeServerError
	}
}

// newHTTPError 根据状态码创建标准化的 HTTPError
func newHTTPError(statusCode int, message string, body []byte) *providers.HTTPError {
	errType, errCode := classifyHTTPStatus(statusCode)
	return &providers.HTTPError{
		ErrorType:     errType,
		ErrorCode:     errCode,
		Message:       message,
		RawStatusCode: statusCode,
		RawBody:       body,
	}
}

// newHTTPErrorf 根据状态码创建格式化消息的 HTTPError
func newHTTPErrorf(statusCode int, body []byte, format string, args ...any) *providers.HTTPError {
	return newHTTPError(statusCode, fmt.Sprintf(format, args...), body)
}
