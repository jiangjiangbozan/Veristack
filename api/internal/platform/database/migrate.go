// Package database 的迁移文件，负责在启动时应用嵌入的 Goose 迁移脚本。
package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"api/migrations"
)

const (
	dialectPostgres = "postgres"
	migrationsDir   = "."
)

// Migrate 在服务启动时执行全部未应用的迁移，保证表结构与代码同步就绪。
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	goose.SetBaseFS(migrations.Files)
	if err := goose.SetDialect(dialectPostgres); err != nil {
		return fmt.Errorf("设置迁移方言失败: %w", err)
	}
	db := stdlib.OpenDBFromPool(pool)
	defer db.Close()
	if err := goose.UpContext(ctx, db, migrationsDir); err != nil {
		return fmt.Errorf("执行数据库迁移失败: %w", err)
	}
	return nil
}
