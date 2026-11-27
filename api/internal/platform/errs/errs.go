// Package errs 定义统一的应用错误，让 Handler 能把内部错误映射为稳定的 HTTP 语义。
package errs

import (
	"errors"
	"net/http"
)

const (
	codeBadRequest    = "bad_request"
	codeForbidden     = "forbidden"
	codeInternal      = "internal_error"
	codeNotFound      = "not_found"
	codeUnauthorized  = "unauthorized"
	codeUnavailable   = "service_unavailable"
	codeTooMany       = "too_many_requests"
	defaultBadRequest = "请求参数无效"
	defaultInternal   = "服务内部错误"
)

// AppError 携带错误码与用户可读信息，Err 保留底层错误供日志记录。
type AppError struct {
	Code   string `json:"code"`    // 稳定错误码
	Err    error  `json:"-"`       // 底层错误
	Status int    `json:"-"`       // HTTP 状态码
	Text   string `json:"message"` // 用户可读信息
}

// Error 实现 error 接口。
func (e *AppError) Error() string {
	return e.Text
}

// Unwrap 暴露底层错误，供 errors.Is 与 errors.As 使用。
func (e *AppError) Unwrap() error {
	return e.Err
}

// BadRequest 构造 400 错误，用于外部输入边界校验失败。
func BadRequest(text string, err error) *AppError {
	return &AppError{Code: codeBadRequest, Err: err, Status: http.StatusBadRequest, Text: text}
}

// Forbidden 构造 403 错误，用于资源归属或权限不足。
func Forbidden(text string, err error) *AppError {
	return &AppError{Code: codeForbidden, Err: err, Status: http.StatusForbidden, Text: text}
}

// Internal 构造 500 错误，内部细节不暴露给客户端。
func Internal(err error) *AppError {
	return &AppError{Code: codeInternal, Err: err, Status: http.StatusInternalServerError, Text: defaultInternal}
}

// NotFound 构造 404 错误，用于资源不存在。
func NotFound(text string, err error) *AppError {
	return &AppError{Code: codeNotFound, Err: err, Status: http.StatusNotFound, Text: text}
}

// Unauthorized 构造 401 错误，用于未登录或凭证失效。
func Unauthorized(text string, err error) *AppError {
	return &AppError{Code: codeUnauthorized, Err: err, Status: http.StatusUnauthorized, Text: text}
}

// Unavailable 构造 503 错误，用于依赖尚未就绪的临时故障。
func Unavailable(text string, err error) *AppError {
	return &AppError{Code: codeUnavailable, Err: err, Status: http.StatusServiceUnavailable, Text: text}
}

// TooManyRequests 构造 429 错误，用于登录等敏感入口的限流。
func TooManyRequests(text string, err error) *AppError {
	return &AppError{Code: codeTooMany, Err: err, Status: http.StatusTooManyRequests, Text: text}
}

// From 将任意错误归一化为 AppError，未知错误按内部错误处理。
func From(err error) *AppError {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	return Internal(err)
}

// BadRequestDefault 返回默认文案的 400 错误。
func BadRequestDefault(err error) *AppError {
	return BadRequest(defaultBadRequest, err)
}
