// Package middleware 提供 Chi 中间件，统一处理鉴权、请求上下文与错误恢复。
package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"api/internal/platform/auth"
	"api/internal/platform/db"
	"api/internal/platform/errs"
	"api/internal/platform/response"
)

const (
	authScheme     = "Bearer"
	contextKeyUser = "authenticatedUser"
)

type contextKey string

// Authenticate 返回校验 Bearer 令牌、核对用户状态并把身份写入请求上下文的中间件。
//
// 每次请求读取用户资料时间戳：密码修改、退出登录或账户停用后，
// 此前签发的令牌会立即失效，不需要在前端维护额外的会话表。
func Authenticate(manager *auth.Manager, pool *pgxpool.Pool) func(http.Handler) http.Handler {
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
			user, err := db.New(pool).GetUserByID(r.Context(), parseUserID(claims.UserID))
			if err != nil {
				response.Fail(w, errs.Unauthorized("用户不存在或已停用", err))
				return
			}
			if user.Status != "active" {
				response.Fail(w, errs.Unauthorized("用户已停用", nil))
				return
			}
			if claims.IssuedAt != nil && user.UpdatedAt.After(claims.IssuedAt.Time.Add(time.Second)) {
				response.Fail(w, errs.Unauthorized("会话已失效，请重新登录", nil))
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

// parseUserID 解析令牌中的用户 ID；格式无效时返回零值，由数据库查询拒绝。
func parseUserID(raw string) uuid.UUID {
	id, _ := uuid.Parse(raw)
	return id
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
