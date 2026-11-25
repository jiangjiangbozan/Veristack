// Package access 从请求身份解析出组织范围，供 Handler 做最小权限校验。
package access

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"

	"api/internal/platform/db"
	"api/internal/platform/errs"
	"api/internal/platform/middleware"
)

// Principal 描述当前请求用户的身份与组织范围。
type Principal struct {
	ID             uuid.UUID // 用户 ID
	OrganizationID uuid.UUID // 所属组织 ID
	Role           string    // 用户角色
}

// FromRequest 从 JWT 声明与数据库读取当前用户，构造访问主体。
func FromRequest(ctx context.Context, r *http.Request, queries *db.Queries) (Principal, error) {
	claims := middleware.UserFrom(r)
	if claims == nil {
		return Principal{}, errs.Unauthorized("未认证", nil)
	}
	id, err := uuid.Parse(claims.UserID)
	if err != nil {
		return Principal{}, errs.Unauthorized("用户标识无效", err)
	}
	user, err := queries.GetUserByID(ctx, id)
	if err != nil {
		return Principal{}, errs.Unauthorized("用户不存在或已禁用", err)
	}
	return Principal{ID: user.ID, OrganizationID: user.OrganizationID, Role: user.Role}, nil
}

// RequireOwner 校验主体是否为组织 owner 或 admin。
func RequireOwner(principal Principal) error {
	if principal.Role != "owner" && principal.Role != "admin" {
		return errs.Forbidden("需要管理员权限", fmt.Errorf("角色 %s 无权操作", principal.Role))
	}
	return nil
}

// RequireSameOrganization 校验资源是否属于主体所在组织。
func RequireSameOrganization(principal Principal, organizationID uuid.UUID) error {
	if principal.OrganizationID != organizationID {
		return errs.Forbidden("资源不属于当前组织", fmt.Errorf("组织不匹配"))
	}
	return nil
}
