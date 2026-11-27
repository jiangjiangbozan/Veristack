// Package main 生成本地演示数据，按外键依赖顺序为各业务表写入约 1000 条真实感数据。
//
// 该命令只面向本地开发库：默认写入 1000 个项目、1000 个用户、1000 个交付版本等，
// 并在同一个事务内完成。执行前会检查 demo 用户标记，避免重复运行造成数据翻倍。
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"api/internal/platform/config"
	"api/internal/platform/database"
)

const (
	defaultRowCount = 1000
	seedTimeout     = 10 * time.Minute
	seedEmailLike   = "demo.%@veristack.local"
)

// main 连接本机数据库并在单个事务内写入演示数据。
func main() {
	count := flag.Int("count", defaultRowCount, "每个业务表的目标行数")
	flag.Parse()
	if *count < 100 {
		fmt.Fprintln(os.Stderr, "count 至少为 100，避免生成无法覆盖页面的演示数据")
		os.Exit(1)
	}
	ctx, cancel := context.WithTimeout(context.Background(), seedTimeout)
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
	if err := run(ctx, pool, *count); err != nil {
		fmt.Fprintf(os.Stderr, "生成演示数据失败: %v\n", err)
		os.Exit(1)
	}
}

// run 在一个事务内生成全部演示数据，失败时整体回滚。
func run(ctx context.Context, pool *pgxpool.Pool, count int) error {
	if err := ensureSeedSlot(ctx, pool); err != nil {
		return err
	}
	mainOrg, ownerID, err := findMainOrganization(ctx, pool)
	if err != nil {
		return err
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	defer tx.Rollback(ctx)
	state := newSeedState(count, mainOrg, ownerID)
	if err := seedAll(ctx, tx, state); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("提交演示数据失败: %w", err)
	}
	printSummary(state)
	return nil
}

// ensureSeedSlot 检查是否已经运行过演示数据，避免重复插入。
func ensureSeedSlot(ctx context.Context, pool *pgxpool.Pool) error {
	var count int64
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM users WHERE email LIKE $1`, seedEmailLike).Scan(&count); err != nil {
		return fmt.Errorf("检查演示数据标记失败: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("数据库中已存在演示用户（%d 个），为避免重复写入已停止；如需重来请使用全新本地库", count)
	}
	return nil
}

// findMainOrganization 优先选择最近有项目的 owner 组织，保证演示项目出现在当前用户的项目列表。
func findMainOrganization(ctx context.Context, pool *pgxpool.Pool) (uuid.UUID, uuid.UUID, error) {
	var organizationID, ownerID uuid.UUID
	err := pool.QueryRow(ctx, `
		SELECT u.organization_id, u.id
		FROM users u
		JOIN projects p ON p.organization_id = u.organization_id
		WHERE u.role = 'owner' AND u.deleted_at IS NULL
		GROUP BY u.organization_id, u.id
		ORDER BY max(p.created_at) DESC
		LIMIT 1`).Scan(&organizationID, &ownerID)
	if err == nil {
		return organizationID, ownerID, nil
	}
	if err := pool.QueryRow(ctx, `
		SELECT organization_id, id FROM users
		WHERE role = 'owner' AND deleted_at IS NULL
		ORDER BY created_at DESC LIMIT 1`).Scan(&organizationID, &ownerID); err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("找不到 owner 用户，请先注册一个组织: %w", err)
	}
	return organizationID, ownerID, nil
}

// printSummary 输出本次写入的关键行数，便于执行后核对。
func printSummary(state *seedState) {
	fmt.Printf("演示数据生成完成：组织 %d、用户 %d、项目 %d、需求 %d、交付 %d、通知 %d。\n",
		len(state.organizationIDs), len(state.users), len(state.projectIDs), len(state.requirementSets)*requirementsPerProject, len(state.deliveryIDs), state.count)
}
