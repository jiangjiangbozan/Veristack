// Package middleware 提供 Chi 中间件，统一处理鉴权、请求上下文与错误恢复。
package middleware

import (
	"context"
	"net/http"
	"strings"

	"api/internal/platform/auth"
	"api/internal/platform/errs"
	"api/internal/platform/response"
)

const (
	authScheme     = "Bearer"
	contextKeyUser = "authenticatedUser"
)

type contextKey string

// Authenticate 返回校验 Bearer 令牌并把身份写入请求上下文的中间件。
func Authenticate(manager *auth.Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := extractToken(r)
			if raw == "" {
				response.Fail(w, errs.Unauthorized("缺少访问令牌", nil))
				return
			}
			claims, err := manager.Parse(raw)
			if err != nil {
				response.Fail(w, errs.Unauthorized("访问令牌无效", err))
				return
			}
			ctx := context.WithValue(r.Context(), contextKey(contextKeyUser), claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserFrom 从请求上下文取出已认证的用户声明。
func UserFrom(r *http.Request) *auth.Claims {
	claims, _ := r.Context().Value(contextKey(contextKeyUser)).(*auth.Claims)
	return claims
}

// extractToken 从 Authorization 头提取 Bearer 令牌，格式不符时返回空串。
func extractToken(r *http.Request) string {
	header := r.Header.Get("Authorization")
	parts := strings.Fields(header)
	if len(parts) != 2 || parts[0] != authScheme {
		return ""
	}
	return parts[1]
}
