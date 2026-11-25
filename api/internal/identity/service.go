// Package identity 的业务服务，负责注册时组织与 owner 用户的原子创建，以及登录认证。
package identity

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"api/internal/platform/auth"
	"api/internal/platform/db"
	"api/internal/platform/errs"
)

const roleOwner = "owner"

// Service 提供身份相关的业务方法。
type Service struct {
	Auth     *auth.Manager // 令牌管理器
	Database *pgxpool.Pool // 数据库连接池
}

// Register 在同一事务中创建组织与 owner 用户，避免留下无用户的空组织。
func (s *Service) Register(ctx context.Context, req RegisterRequest) (db.User, string, error) {
	tx, err := s.Database.Begin(ctx)
	if err != nil {
		return db.User{}, "", fmt.Errorf("开启注册事务失败: %w", err)
	}
	defer tx.Rollback(ctx)
	queries := db.New(tx)
	organization, err := queries.CreateOrganization(ctx, req.OrganizationName)
	if err != nil {
		return db.User{}, "", fmt.Errorf("创建组织失败: %w", err)
	}
	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		return db.User{}, "", fmt.Errorf("哈希密码失败: %w", err)
	}
	user, err := queries.CreateUser(ctx, db.CreateUserParams{
		DisplayName:    req.DisplayName,
		Email:          req.Email,
		OrganizationID: organization.ID,
		PasswordHash:   passwordHash,
		Role:           roleOwner,
	})
	if err != nil {
		return db.User{}, "", fmt.Errorf("创建用户失败: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return db.User{}, "", fmt.Errorf("提交注册事务失败: %w", err)
	}
	token, err := s.Auth.Generate(user.ID.String(), user.Role)
	if err != nil {
		return db.User{}, "", fmt.Errorf("签发令牌失败: %w", err)
	}
	return user, token, nil
}

// Login 校验邮箱与密码，成功时签发访问令牌。
func (s *Service) Login(ctx context.Context, req LoginRequest) (db.User, string, error) {
	user, err := db.New(s.Database).GetUserByEmail(ctx, req.Email)
	if err != nil {
		return db.User{}, "", errs.Unauthorized("邮箱或密码错误", err)
	}
	if user.Status != "active" || !auth.VerifyPassword(user.PasswordHash, req.Password) {
		return db.User{}, "", errs.Unauthorized("邮箱或密码错误", nil)
	}
	token, err := s.Auth.Generate(user.ID.String(), user.Role)
	if err != nil {
		return db.User{}, "", fmt.Errorf("签发令牌失败: %w", err)
	}
	return user, token, nil
}

// GetUser 按 ID 读取当前用户，供 /auth/me 接口使用。
func (s *Service) GetUser(ctx context.Context, id uuid.UUID) (db.User, error) {
	return db.New(s.Database).GetUserByID(ctx, id)
}
