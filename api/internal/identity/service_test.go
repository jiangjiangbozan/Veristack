// Package identity 的业务逻辑单元测试，使用内存替身隔离数据库与网络。
package identity

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"api/internal/platform/auth"
	"api/internal/platform/db"
	"api/internal/platform/errs"
	"api/internal/platform/outbox"
)

type fakeUserStore struct {
	code           db.EmailVerificationCode
	codeErr        error
	consumed       bool
	createdParams  db.CreateEmailVerificationCodeParams
	incrementCalls int
	latestCode     db.EmailVerificationCode
	latestErr      error
	markedVerified bool
	userByEmail    db.User
	userByEmailErr error
	userByID       db.User
	userByIDErr    error
}

// ConsumeEmailVerificationCode 记录验证码消费。
func (f *fakeUserStore) ConsumeEmailVerificationCode(_ context.Context, _ uuid.UUID) (db.EmailVerificationCode, error) {
	f.consumed = true
	return f.code, f.codeErr
}

// CreateEmailVerificationCode 记录验证码创建参数。
func (f *fakeUserStore) CreateEmailVerificationCode(_ context.Context, arg db.CreateEmailVerificationCodeParams) (db.EmailVerificationCode, error) {
	f.createdParams = arg
	return f.code, f.codeErr
}

// GetLatestEmailVerificationCodeByEmail 返回预置的最新验证码。
func (f *fakeUserStore) GetLatestEmailVerificationCodeByEmail(_ context.Context, _ string) (db.EmailVerificationCode, error) {
	return f.latestCode, f.latestErr
}

// GetUserByEmail 返回预置用户。
func (f *fakeUserStore) GetUserByEmail(_ context.Context, _ string) (db.User, error) {
	return f.userByEmail, f.userByEmailErr
}

// GetUserByID 返回预置用户。
func (f *fakeUserStore) GetUserByID(_ context.Context, _ uuid.UUID) (db.User, error) {
	return f.userByID, f.userByIDErr
}

// IncrementEmailVerificationAttempts 记录错误尝试次数。
func (f *fakeUserStore) IncrementEmailVerificationAttempts(_ context.Context, _ uuid.UUID) error {
	f.incrementCalls++
	return nil
}

// MarkUserEmailVerified 记录邮箱验证成功。
func (f *fakeUserStore) MarkUserEmailVerified(_ context.Context, _ uuid.UUID) (db.User, error) {
	f.markedVerified = true
	return f.userByID, nil
}

type fakeAccountStore struct {
	createdTokenHash string
	createdUserID    uuid.UUID
	resetCalled      bool
	resetErr         error
	touched          bool
	updateErr        error
	updatedHash      string
	validToken       bool
	validTokenErr    error
}

// CreatePasswordResetToken 记录重置令牌。
func (f *fakeAccountStore) CreatePasswordResetToken(_ context.Context, userID uuid.UUID, tokenHash string, _ time.Time) error {
	f.createdUserID = userID
	f.createdTokenHash = tokenHash
	return nil
}

// HasValidPasswordResetToken 返回预置令牌有效性。
func (f *fakeAccountStore) HasValidPasswordResetToken(_ context.Context, _ string) (bool, error) {
	return f.validToken, f.validTokenErr
}

// ResetPassword 记录重置密码调用。
func (f *fakeAccountStore) ResetPassword(_ context.Context, _, _ string) error {
	f.resetCalled = true
	return f.resetErr
}

// TouchUser 记录会话失效调用。
func (f *fakeAccountStore) TouchUser(_ context.Context, _ uuid.UUID) error {
	f.touched = true
	return nil
}

// UpdatePassword 记录密码摘要更新。
func (f *fakeAccountStore) UpdatePassword(_ context.Context, _ uuid.UUID, passwordHash string) error {
	f.updatedHash = passwordHash
	return f.updateErr
}

type fakeMailer struct {
	body    string
	subject string
	to      string
}

// Send 记录同步邮件。
func (f *fakeMailer) Send(_ context.Context, to, subject, body string) error {
	f.to = to
	f.subject = subject
	f.body = body
	return nil
}

type fakeEmailQueue struct {
	dedupeKey string
	payload   outbox.EmailPayload
}

// EnqueueEmail 记录异步邮件任务。
func (f *fakeEmailQueue) EnqueueEmail(_ context.Context, payload outbox.EmailPayload, dedupeKey string) error {
	f.payload = payload
	f.dedupeKey = dedupeKey
	return nil
}

func TestChangePassword_RejectsWrongCurrentPassword(t *testing.T) {
	passwordHash, err := auth.HashPassword("old-password")
	if err != nil {
		t.Fatalf("哈希测试密码失败: %v", err)
	}
	userID := uuid.New()
	service := Service{
		Accounts: &fakeAccountStore{},
		Users:    &fakeUserStore{userByID: db.User{ID: userID, Status: "active", PasswordHash: passwordHash}},
	}

	err = service.ChangePassword(context.Background(), userID, "wrong-password", "new-password")

	if err == nil {
		t.Fatalf("当前密码错误时应拒绝修改")
	}
	if code := errs.From(err).Code; code != "bad_request" {
		t.Fatalf("错误码应为 bad_request，实际为 %s", code)
	}
}

func TestChangePassword_UpdatesHash(t *testing.T) {
	passwordHash, err := auth.HashPassword("old-password")
	if err != nil {
		t.Fatalf("哈希测试密码失败: %v", err)
	}
	accounts := &fakeAccountStore{}
	userID := uuid.New()
	service := Service{
		Accounts: accounts,
		Users:    &fakeUserStore{userByID: db.User{ID: userID, Status: "active", PasswordHash: passwordHash}},
	}

	if err := service.ChangePassword(context.Background(), userID, "old-password", "new-password"); err != nil {
		t.Fatalf("修改密码失败: %v", err)
	}
	if !auth.VerifyPassword(accounts.updatedHash, "new-password") {
		t.Fatalf("新密码摘要不正确")
	}
}

func TestRequestPasswordReset_UnknownEmailDoesNothing(t *testing.T) {
	accounts := &fakeAccountStore{}
	queue := &fakeEmailQueue{}
	service := Service{
		Accounts:   accounts,
		EmailQueue: queue,
		Users:      &fakeUserStore{userByEmailErr: pgx.ErrNoRows},
	}

	if err := service.RequestPasswordReset(context.Background(), "missing@example.com"); err != nil {
		t.Fatalf("未知邮箱应静默返回: %v", err)
	}
	if accounts.createdTokenHash != "" || queue.dedupeKey != "" {
		t.Fatalf("未知邮箱不应创建令牌或邮件任务")
	}
}

func TestRequestPasswordReset_EnqueuesEmailWithToken(t *testing.T) {
	accounts := &fakeAccountStore{}
	queue := &fakeEmailQueue{}
	service := Service{
		Accounts:      accounts,
		EmailQueue:    queue,
		Users:         &fakeUserStore{userByEmail: db.User{ID: uuid.New(), Email: "user@example.com", Status: "active"}},
		PublicBaseURL: "https://veristack.example.com",
	}

	if err := service.RequestPasswordReset(context.Background(), "user@example.com"); err != nil {
		t.Fatalf("申请重置密码失败: %v", err)
	}
	if accounts.createdTokenHash == "" {
		t.Fatalf("应保存重置令牌摘要")
	}
	if queue.dedupeKey != "password-reset:"+accounts.createdTokenHash {
		t.Fatalf("邮件去重键不正确: %q", queue.dedupeKey)
	}
	if queue.payload.To != "user@example.com" || queue.payload.Subject != "重置 Veristack 密码" {
		t.Fatalf("邮件收件人或主题不正确: %+v", queue.payload)
	}
	if !contains(queue.payload.Body, "https://veristack.example.com/auth/reset/") {
		t.Fatalf("邮件正文应包含公网重置链接: %q", queue.payload.Body)
	}
}

func TestRequestEmailVerificationCode_EnqueuesEmail(t *testing.T) {
	codeID := uuid.New()
	users := &fakeUserStore{
		code:        db.EmailVerificationCode{ID: codeID},
		userByID:    db.User{ID: uuid.New(), Email: "user@example.com", Status: "active"},
		userByEmail: db.User{},
	}
	queue := &fakeEmailQueue{}
	service := Service{EmailQueue: queue, Users: users}

	if err := service.RequestEmailVerificationCode(context.Background(), uuid.New()); err != nil {
		t.Fatalf("申请邮箱验证码失败: %v", err)
	}
	if users.createdParams.Email != "user@example.com" {
		t.Fatalf("验证码应写入用户邮箱，实际为 %q", users.createdParams.Email)
	}
	if queue.dedupeKey != "email-code:"+codeID.String() {
		t.Fatalf("邮件去重键不正确: %q", queue.dedupeKey)
	}
	if len(queue.payload.Body) == 0 || !contains(queue.payload.Body, "验证码") {
		t.Fatalf("邮件正文应包含验证码提示: %q", queue.payload.Body)
	}
}

func TestVerifyEmail_WrongCodeIncrementsAttempts(t *testing.T) {
	userID := uuid.New()
	users := &fakeUserStore{
		userByID: db.User{ID: userID, Email: "user@example.com", Status: "active"},
		latestCode: db.EmailVerificationCode{
			ID:       uuid.New(),
			CodeHash: hashToken("123456"),
		},
	}
	service := Service{Users: users}

	_, err := service.VerifyEmail(context.Background(), userID, "000000")

	if err == nil {
		t.Fatalf("错误验证码应拒绝")
	}
	if users.incrementCalls != 1 {
		t.Fatalf("错误验证码应累计一次尝试，实际为 %d", users.incrementCalls)
	}
	if users.consumed {
		t.Fatalf("错误验证码不应被消费")
	}
}

func TestVerifyEmail_RejectsTooManyAttempts(t *testing.T) {
	userID := uuid.New()
	users := &fakeUserStore{
		userByID: db.User{ID: userID, Email: "user@example.com", Status: "active"},
		latestCode: db.EmailVerificationCode{
			ID:       uuid.New(),
			Attempts: verificationCodeMaxAttempts,
			CodeHash: hashToken("123456"),
		},
	}
	service := Service{Users: users}

	_, err := service.VerifyEmail(context.Background(), userID, "123456")

	if err == nil {
		t.Fatalf("尝试次数达到上限后应拒绝")
	}
	if users.consumed || users.incrementCalls != 0 {
		t.Fatalf("达到上限后不应再消费或累计尝试")
	}
}

func TestVerifyEmail_ConsumesAndMarksVerified(t *testing.T) {
	userID := uuid.New()
	users := &fakeUserStore{
		userByID: db.User{ID: userID, Email: "user@example.com", Status: "active"},
		latestCode: db.EmailVerificationCode{
			ID:       uuid.New(),
			CodeHash: hashToken("123456"),
		},
	}
	service := Service{Users: users}

	verified, err := service.VerifyEmail(context.Background(), userID, "123456")

	if err != nil {
		t.Fatalf("正确验证码应通过: %v", err)
	}
	if !users.consumed || !users.markedVerified {
		t.Fatalf("正确验证码应消费并标记邮箱已验证")
	}
	if verified.ID != userID {
		t.Fatalf("应返回验证后的用户，实际 ID 为 %s", verified.ID)
	}
}

func TestResetPassword_RejectsInvalidToken(t *testing.T) {
	accounts := &fakeAccountStore{validToken: false}
	service := Service{Accounts: accounts}

	err := service.ResetPassword(context.Background(), "invalid-token", "new-password")

	if err == nil {
		t.Fatalf("无效重置令牌应拒绝")
	}
	if accounts.resetCalled {
		t.Fatalf("无效令牌不应执行密码重置")
	}
}

func TestResetLink_UsesFallbackBaseURL(t *testing.T) {
	service := Service{}
	if link := service.resetLink("abc"); link != "http://localhost:5173/auth/reset/abc" {
		t.Fatalf("未配置公网地址时应使用本地默认地址，实际为 %q", link)
	}
}

func contains(value, substring string) bool {
	return strings.Contains(value, substring)
}
