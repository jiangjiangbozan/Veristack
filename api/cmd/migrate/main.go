// Package main 执行数据库迁移，不启动 HTTP 服务。
//
// 本地 Docker 数据库只需设置 VERISTACK_DATABASE_DSN 后运行本命令，
// 不必为了迁移启动完整 API，也不会因令牌密钥等无关配置缺失而失败。
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"api/internal/platform/config"
	"api/internal/platform/database"
)

const (
	migrationTimeout = 2 * time.Minute
	versionQuery     = `SELECT COALESCE(MAX(version_id), 0) FROM goose_db_version WHERE is_applied`
)

// main 加载数据库配置、执行全部未应用迁移并输出结果。
func main() {
	ctx, cancel := context.WithTimeout(context.Background(), migrationTimeout)
	defer cancel()

	cfg, err := config.LoadDatabase()
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载数据库配置失败: %v\n", err)
		os.Exit(1)
	}
	pool, err := database.New(ctx, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "连接数据库失败: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()
	if err := database.Migrate(ctx, pool); err != nil {
		fmt.Fprintf(os.Stderr, "执行数据库迁移失败: %v\n", err)
		os.Exit(1)
	}
	var version int64
	if err := pool.QueryRow(ctx, versionQuery).Scan(&version); err != nil {
		fmt.Fprintf(os.Stderr, "读取迁移版本失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("数据库迁移完成，当前版本 %d\n", version)
}
