// Package platform 的审计辅助，把「业务写入 + 审计写入」封装成一次原子操作。
package platform

import (
	"context"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"api/internal/platform/access"
	"api/internal/platform/database"
)

// Audited 在事务中执行业务写入并追加审计事件。
//
// 业务写入与审计写入共享同一事务：审计失败会使业务变更一并回滚，
// 不会出现「界面状态已更新，但系统说不清发生过什么」。
func Audited[T any](
	ctx context.Context,
	pool *pgxpool.Pool,
	r *http.Request,
	principal access.Principal,
	write func(ctx context.Context) (T, error),
	audit func(ctx context.Context, result T) error,
) (T, error) {
	var result T
	err := database.WithTx(ctx, pool, func(txContext context.Context) error {
		value, err := write(txContext)
		if err != nil {
			return err
		}
		result = value
		return audit(txContext, value)
	})
	return result, err
}
