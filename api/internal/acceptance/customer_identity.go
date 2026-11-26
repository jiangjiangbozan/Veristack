// Package acceptance 的客户身份验证服务，确保验收结论归属于可追溯的客户身份。
package acceptance

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"api/internal/platform/database"
	"api/internal/platform/datetime"
	"api/internal/platform/db"
	"api/internal/platform/errs"
)

const (
	// verificationCodeLength 是邮箱验证码位数。
	verificationCodeLength = 6
	// verificationCodeTTL 是验证码有效期。
	verificationCodeTTL = 15 * time.Minute
	// maxVerificationAttempts 限制单个验证码的尝试次数。
	maxVerificationAttempts = 5
)

// CustomerIdentityService 负责客户邮箱验证与验收身份登记。
type CustomerIdentityService struct {
	Database *pgxpool.Pool // 数据库连接池
	Mailer   Mailer        // 邮件发送器
}

// Mailer 抽象邮件发送能力，便于在未配置 SMTP 时以日志替代。
type Mailer interface {
	SendVerificationCode(ctx context.Context, email, code string) error
}

// LogMailer 在未配置邮件通道时把验证码写入日志，仅用于本地开发。
type LogMailer struct{}

// SendVerificationCode 把验证码写入标准输出，便于本地联调。
func (LogMailer) SendVerificationCode(_ context.Context, email, code string) error {
	fmt.Printf("验证码 %s 已发送至 %s\n", code, email)
	return nil
}

// RequestVerificationCode 为客户邮箱签发一次性验证码。
func (s *CustomerIdentityService) RequestVerificationCode(ctx context.Context, email string) error {
	code, err := generateVerificationCode()
	if err != nil {
		return errs.Internal(err)
	}
	_, err = database.Queries(ctx, s.Database).CreateEmailVerificationCode(ctx, db.CreateEmailVerificationCodeParams{
		CodeHash:  hashVerificationCode(code),
		Email:     strings.ToLower(strings.TrimSpace(email)),
		ExpiresAt: datetime.Now().Add(verificationCodeTTL),
	})
	if err != nil {
		return errs.Internal(err)
	}
	if err := s.Mailer.SendVerificationCode(ctx, email, code); err != nil {
		return errs.Internal(err)
	}
	return nil
}

// VerifyCode 核验客户提交的验证码，并标记邮箱已验证。
func (s *CustomerIdentityService) VerifyCode(ctx context.Context, email, code string) error {
	queries := database.Queries(ctx, s.Database)
	record, err := queries.GetEmailVerificationCode(ctx, hashVerificationCode(code))
	if err != nil {
		return errs.BadRequest("验证码无效或已过期", nil)
	}
	if record.Attempts >= maxVerificationAttempts {
		return errs.BadRequest("验证码尝试次数过多，请重新获取", nil)
	}
	if _, err := queries.ConsumeEmailVerificationCode(ctx, record.ID); err != nil {
		return errs.BadRequest("验证码已被使用", nil)
	}
	user, err := queries.GetUserByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if err == nil {
		if _, err := queries.MarkUserEmailVerified(ctx, user.ID); err != nil {
			return errs.Internal(err)
		}
	}
	return nil
}

// generateVerificationCode 生成数字验证码。
func generateVerificationCode() (string, error) {
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

// hashVerificationCode 计算验证码摘要，数据库只保存摘要不保存明文。
func hashVerificationCode(code string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(code)))
	return hex.EncodeToString(sum[:])
}
