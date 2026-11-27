// Package identity 的数据访问层，封装账户安全相关的原生 SQL。
package identity

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"api/internal/platform/database"
	"api/internal/platform/db"
)

// AccountStore 抽象账户安全写操作，便于 Service 在单元测试中注入替身。
type AccountStore interface {
	CreatePasswordResetToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error
	HasValidPasswordResetToken(ctx context.Context, tokenHash string) (bool, error)
	ResetPassword(ctx context.Context, tokenHash, passwordHash string) error
	TouchUser(ctx context.Context, userID uuid.UUID) error
	UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error
}

// UserStore 抽象用户与验证码数据访问，查询默认参与请求级事务。
type UserStore interface {
	ConsumeEmailVerificationCode(ctx context.Context, id uuid.UUID) (db.EmailVerificationCode, error)
	CreateEmailVerificationCode(ctx context.Context, arg db.CreateEmailVerificationCodeParams) (db.EmailVerificationCode, error)
	GetLatestEmailVerificationCodeByEmail(ctx context.Context, email string) (db.EmailVerificationCode, error)
	GetUserByEmail(ctx context.Context, email string) (db.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (db.User, error)
	IncrementEmailVerificationAttempts(ctx context.Context, id uuid.UUID) error
	MarkUserEmailVerified(ctx context.Context, id uuid.UUID) (db.User, error)
}

// Repository 负责账户安全相关的数据库写入。
type Repository struct {
	Database *pgxpool.Pool // 数据库连接池
}

// NewRepository 创建账户数据访问对象。
func NewRepository(database *pgxpool.Pool) *Repository {
	return &Repository{Database: database}
}

// UpdatePassword 更新密码摘要，并推进 updated_at 使旧会话令牌失效。
func (r *Repository) UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	tag, err := database.Executor(ctx, r.Database).Exec(ctx, `
		UPDATE users
		SET password_hash = $2, updated_at = now()
		WHERE id = $1 AND status = 'active' AND deleted_at IS NULL
	`, userID, passwordHash)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// CreatePasswordResetToken 保存一次性重置令牌摘要。
func (r *Repository) CreatePasswordResetToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error {
	_, err := database.Executor(ctx, r.Database).Exec(ctx, `
		INSERT INTO password_reset_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
	`, userID, tokenHash, expiresAt)
	return err
}

// HasValidPasswordResetToken 判断令牌当前是否可用。
//
// 先做廉价的存在性检查，避免无效令牌也触发密码哈希，
// 让公开的重置接口不能成为 CPU 消耗入口。
func (r *Repository) HasValidPasswordResetToken(ctx context.Context, tokenHash string) (bool, error) {
	var exists bool
	err := database.Executor(ctx, r.Database).QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM password_reset_tokens prt
			JOIN users u ON u.id = prt.user_id
			WHERE prt.token_hash = $1 AND prt.consumed_at IS NULL
			  AND prt.expires_at > now()
			  AND u.status = 'active' AND u.deleted_at IS NULL
		)
	`, tokenHash).Scan(&exists)
	return exists, err
}

// ResetPassword 原子消费重置令牌并更新密码。
//
// 令牌消费与密码更新必须同生共死：否则密码更新失败时令牌已被烧掉，
// 用户会拿到一个“链接无效”却无法重试的死胡同。
func (r *Repository) ResetPassword(ctx context.Context, tokenHash, passwordHash string) error {
	tag, err := database.Executor(ctx, r.Database).Exec(ctx, `
		WITH consumed AS (
			UPDATE password_reset_tokens AS prt
			SET consumed_at = now()
			WHERE prt.token_hash = $1 AND prt.consumed_at IS NULL AND prt.expires_at > now()
			  AND EXISTS (
				SELECT 1 FROM users u
				WHERE u.id = prt.user_id
				  AND u.status = 'active' AND u.deleted_at IS NULL
			  )
			RETURNING prt.user_id
		)
		UPDATE users
		SET password_hash = $2, updated_at = now()
		FROM consumed
		WHERE users.id = consumed.user_id
	`, tokenHash, passwordHash)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// TouchUser 推进用户资料时间戳，使此前签发的会话令牌全部失效。
func (r *Repository) TouchUser(ctx context.Context, userID uuid.UUID) error {
	tag, err := database.Executor(ctx, r.Database).Exec(ctx, `
		UPDATE users SET updated_at = now() WHERE id = $1 AND deleted_at IS NULL
	`, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// sqlUserStore 是 UserStore 的生产实现。
type sqlUserStore struct {
	Database *pgxpool.Pool // 数据库连接池
}

// NewUserStore 创建用户与验证码数据访问对象。
func NewUserStore(pool *pgxpool.Pool) UserStore {
	return &sqlUserStore{Database: pool}
}

// ConsumeEmailVerificationCode 标记验证码已被使用。
func (s *sqlUserStore) ConsumeEmailVerificationCode(ctx context.Context, id uuid.UUID) (db.EmailVerificationCode, error) {
	return database.Queries(ctx, s.Database).ConsumeEmailVerificationCode(ctx, id)
}

// CreateEmailVerificationCode 保存一次性邮箱验证码。
func (s *sqlUserStore) CreateEmailVerificationCode(ctx context.Context, arg db.CreateEmailVerificationCodeParams) (db.EmailVerificationCode, error) {
	return database.Queries(ctx, s.Database).CreateEmailVerificationCode(ctx, arg)
}

// GetLatestEmailVerificationCodeByEmail 读取邮箱最近一条待验证的验证码。
func (s *sqlUserStore) GetLatestEmailVerificationCodeByEmail(ctx context.Context, email string) (db.EmailVerificationCode, error) {
	return database.Queries(ctx, s.Database).GetLatestEmailVerificationCodeByEmail(ctx, email)
}

// GetUserByEmail 按邮箱读取用户。
func (s *sqlUserStore) GetUserByEmail(ctx context.Context, email string) (db.User, error) {
	return database.Queries(ctx, s.Database).GetUserByEmail(ctx, email)
}

// GetUserByID 按 ID 读取用户。
func (s *sqlUserStore) GetUserByID(ctx context.Context, id uuid.UUID) (db.User, error) {
	return database.Queries(ctx, s.Database).GetUserByID(ctx, id)
}

// IncrementEmailVerificationAttempts 记录错误尝试次数。
//
// 错误验证码会让 Handler 返回 400，请求级事务随之回滚；
// 因此这里故意使用连接池独立写入，确保失败次数不会因为事务回滚被抹掉。
func (s *sqlUserStore) IncrementEmailVerificationAttempts(ctx context.Context, id uuid.UUID) error {
	return db.New(s.Database).IncrementEmailVerificationAttempts(ctx, id)
}

// MarkUserEmailVerified 标记用户邮箱已验证。
func (s *sqlUserStore) MarkUserEmailVerified(ctx context.Context, id uuid.UUID) (db.User, error) {
	return database.Queries(ctx, s.Database).MarkUserEmailVerified(ctx, id)
}

// 编译期断言：Repository 必须满足 AccountStore。
var _ AccountStore = (*Repository)(nil)
