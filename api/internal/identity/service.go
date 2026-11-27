// Package identity 的业务服务，负责注册时组织与 owner 用户的原子创建，以及登录认证。
package identity

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"api/internal/platform/auth"
	"api/internal/platform/database"
	"api/internal/platform/datetime"
	"api/internal/platform/db"
	"api/internal/platform/errs"
	"api/internal/platform/mailer"
	"api/internal/platform/outbox"
)

const (
	roleOwner                   = "owner"
	verificationCodeLength      = 6
	verificationCodeMaxAttempts = 5
	verificationCodeTTL         = 15 * time.Minute
	resetTokenLength            = 32
	resetTokenTTL               = 30 * time.Minute
)

// Service 提供身份相关的业务方法。
type Service struct {
	Accounts      AccountStore      // 账户写操作，可为空
	Auth          *auth.Manager     // 令牌管理器
	Database      *pgxpool.Pool     // 数据库连接池
	EmailQueue    outbox.EmailQueue // 邮件任务队列，未配置时回退为同步发送
	Mailer        mailer.Mailer     // 邮件发送器
	PublicBaseURL string            // 前端公网地址，用于拼接邮件链接
	Repository    *Repository       // 账户安全数据访问
	Users         UserStore         // 用户与验证码数据访问，可为空
}

// Register 在同一事务中创建组织与 owner 用户，避免留下无用户的空组织。
func (s *Service) Register(ctx context.Context, req RegisterRequest) (db.User, string, error) {
	queries := database.Queries(ctx, s.Database)
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
	token, err := s.Auth.Generate(user.ID.String(), user.Role)
	if err != nil {
		return db.User{}, "", fmt.Errorf("签发令牌失败: %w", err)
	}
	return user, token, nil
}

// Login 校验邮箱与密码，成功时签发访问令牌。
func (s *Service) Login(ctx context.Context, req LoginRequest) (db.User, string, error) {
	user, err := s.userStore().GetUserByEmail(ctx, req.Email)
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
	return s.userStore().GetUserByID(ctx, id)
}

// ChangePassword 校验当前密码后更新密码，并让此前签发的会话令牌失效。
func (s *Service) ChangePassword(ctx context.Context, id uuid.UUID, current, next string) error {
	user, err := s.GetUser(ctx, id)
	if err != nil {
		return errs.NotFound("用户不存在", err)
	}
	if !auth.VerifyPassword(user.PasswordHash, current) {
		return errs.BadRequest("当前密码不正确", nil)
	}
	passwordHash, err := auth.HashPassword(next)
	if err != nil {
		return fmt.Errorf("哈希密码失败: %w", err)
	}
	if err := s.accountRepository().UpdatePassword(ctx, id, passwordHash); err != nil {
		if err == pgx.ErrNoRows {
			return errs.NotFound("用户不存在或已停用", nil)
		}
		return errs.Internal(err)
	}
	return nil
}

// RequestPasswordReset 为已注册邮箱签发一次性重置令牌。
//
// 无论邮箱是否存在都返回成功，避免通过接口枚举组织成员。
func (s *Service) RequestPasswordReset(ctx context.Context, email string) error {
	user, err := s.userStore().GetUserByEmail(ctx, strings.TrimSpace(email))
	if err != nil || user.Status != "active" {
		return nil
	}
	token, err := generateToken(resetTokenLength)
	if err != nil {
		return errs.Internal(err)
	}
	if err := s.accountRepository().CreatePasswordResetToken(ctx, user.ID, hashToken(token), datetime.Now().Add(resetTokenTTL)); err != nil {
		return errs.Internal(err)
	}
	body := fmt.Sprintf("你正在重置 Veristack 密码。请在 30 分钟内打开链接完成重置：\n\n%s\n\n如果不是你本人操作，请忽略这封邮件。", s.resetLink(token))
	if err := s.enqueueEmail(ctx, user.Email, "重置 Veristack 密码", body, "password-reset:"+hashToken(token)); err != nil {
		return errs.Internal(err)
	}
	return nil
}

// ResetPassword 消费一次性令牌并设置新密码。
func (s *Service) ResetPassword(ctx context.Context, token, next string) error {
	tokenHash := hashToken(strings.TrimSpace(token))
	valid, err := s.accountRepository().HasValidPasswordResetToken(ctx, tokenHash)
	if err != nil {
		return errs.Internal(err)
	}
	if !valid {
		return errs.BadRequest("重置链接无效或已过期", nil)
	}
	passwordHash, err := auth.HashPassword(next)
	if err != nil {
		return fmt.Errorf("哈希密码失败: %w", err)
	}
	if err := s.accountRepository().ResetPassword(ctx, tokenHash, passwordHash); err != nil {
		if err == pgx.ErrNoRows {
			return errs.BadRequest("重置链接无效或已过期", nil)
		}
		return errs.Internal(err)
	}
	return nil
}

// RequestEmailVerificationCode 为当前用户邮箱签发验证码。
func (s *Service) RequestEmailVerificationCode(ctx context.Context, userID uuid.UUID) error {
	user, err := s.GetUser(ctx, userID)
	if err != nil {
		return errs.NotFound("用户不存在", err)
	}
	code, err := generateCode()
	if err != nil {
		return errs.Internal(err)
	}
	record, err := s.userStore().CreateEmailVerificationCode(ctx, db.CreateEmailVerificationCodeParams{
		CodeHash:  hashToken(code),
		Email:     user.Email,
		ExpiresAt: datetime.Now().Add(verificationCodeTTL),
	})
	if err != nil {
		return errs.Internal(err)
	}
	body := fmt.Sprintf("你的邮箱验证码是 %s，15 分钟内有效。", code)
	if err := s.enqueueEmail(ctx, user.Email, "验证你的 Veristack 邮箱", body, "email-code:"+record.ID.String()); err != nil {
		return errs.Internal(err)
	}
	return nil
}

// VerifyEmail 核验验证码并把用户邮箱标记为已验证。
func (s *Service) VerifyEmail(ctx context.Context, userID uuid.UUID, code string) (db.User, error) {
	user, err := s.GetUser(ctx, userID)
	if err != nil {
		return db.User{}, errs.NotFound("用户不存在", err)
	}
	users := s.userStore()
	record, err := users.GetLatestEmailVerificationCodeByEmail(ctx, user.Email)
	if err != nil {
		return db.User{}, errs.BadRequest("验证码无效或已过期", nil)
	}
	if record.Attempts >= verificationCodeMaxAttempts {
		return db.User{}, errs.BadRequest("验证码尝试次数过多，请重新获取", nil)
	}
	expected := hashToken(strings.TrimSpace(code))
	if subtle.ConstantTimeCompare([]byte(record.CodeHash), []byte(expected)) != 1 {
		// 错误尝试会让 Handler 返回 400，请求级事务随之回滚；
		// 这里用连接池单独写入计数，确保失败次数不会因为事务回滚被抹掉。
		if err := users.IncrementEmailVerificationAttempts(ctx, record.ID); err != nil {
			return db.User{}, errs.Internal(err)
		}
		return db.User{}, errs.BadRequest("验证码无效或已过期", nil)
	}
	if _, err := users.ConsumeEmailVerificationCode(ctx, record.ID); err != nil {
		return db.User{}, errs.BadRequest("验证码已被使用", nil)
	}
	verified, err := users.MarkUserEmailVerified(ctx, userID)
	if err != nil {
		return db.User{}, errs.Internal(err)
	}
	return verified, nil
}

// Logout 推进用户资料时间戳，使当前及此前签发的会话令牌全部失效。
func (s *Service) Logout(ctx context.Context, userID uuid.UUID) error {
	if err := s.accountRepository().TouchUser(ctx, userID); err != nil {
		if err == pgx.ErrNoRows {
			return errs.NotFound("用户不存在或已停用", nil)
		}
		return errs.Internal(err)
	}
	return nil
}

// resetLink 拼接前端重置密码地址；未配置公网地址时退化为本地开发地址。
func (s *Service) resetLink(token string) string {
	base := strings.TrimRight(s.PublicBaseURL, "/")
	if base == "" {
		base = "http://localhost:5173"
	}
	return fmt.Sprintf("%s/auth/reset/%s", base, token)
}

// accountRepository 延迟创建数据访问对象，兼容未显式注入 Repository 的调用方。
func (s *Service) accountRepository() AccountStore {
	if s.Accounts != nil {
		return s.Accounts
	}
	if s.Repository == nil {
		s.Repository = NewRepository(s.Database)
	}
	return s.Repository
}

// userStore 返回用户数据访问对象，未显式注入时使用连接池实现。
func (s *Service) userStore() UserStore {
	if s.Users != nil {
		return s.Users
	}
	return NewUserStore(s.Database)
}

// enqueueEmail 优先把邮件写入 Outbox；未配置队列时保留同步发送作为测试与降级路径。
func (s *Service) enqueueEmail(ctx context.Context, to, subject, body, dedupeKey string) error {
	if s.EmailQueue != nil {
		return s.EmailQueue.EnqueueEmail(ctx, outbox.EmailPayload{
			Body:    body,
			Subject: subject,
			To:      to,
		}, dedupeKey)
	}
	return s.mailSender().Send(ctx, to, subject, body)
}

// mailSender 返回邮件发送器，未配置时退化为日志输出。
func (s *Service) mailSender() mailer.Mailer {
	if s.Mailer == nil {
		return mailer.LogMailer{}
	}
	return s.Mailer
}

// generateToken 生成指定字节长度的随机令牌。
func generateToken(length int) (string, error) {
	buffer := make([]byte, length)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("生成令牌失败: %w", err)
	}
	return hex.EncodeToString(buffer), nil
}

// generateCode 生成数字验证码。
func generateCode() (string, error) {
	buffer := make([]byte, verificationCodeLength)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("生成验证码失败: %w", err)
	}
	digits := make([]byte, verificationCodeLength)
	for index, value := range buffer {
		digits[index] = '0' + value%10
	}
	return string(digits), nil
}

// hashToken 计算令牌或验证码的 SHA-256 摘要。
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
