// Package acceptance 的业务服务，负责分享链接、逐条验收裁定、客户确认与异议闭环。
package acceptance

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"api/internal/platform/auth"
	"api/internal/platform/database"
	"api/internal/platform/db"
	"api/internal/platform/dbutil"
	"api/internal/platform/errs"
	"api/internal/platform/views"
)

const (
	decisionApprove = "approve"
	decisionReject  = "reject"
	defaultExpiry   = 14 * 24 * time.Hour
	roleCustomer    = "customer"
	statusApproved  = "approved"
	statusRejected  = "rejected"
)

// Service 提供验收相关的业务方法。
type Service struct {
	Database *pgxpool.Pool // 数据库连接池
}

// CountOpenObjections 统计交付版本中待处理的异议数量。
func (s *Service) CountOpenObjections(ctx context.Context, deliveryID uuid.UUID) (int64, error) {
	return database.Queries(ctx, s.Database).CountOpenObjections(ctx, deliveryID)
}

// CreateShareToken 为交付生成客户分享链接，库中只保存令牌摘要。
func (s *Service) CreateShareToken(ctx context.Context, deliveryID uuid.UUID, expiresInDays int) (string, time.Time, error) {
	raw, err := generateToken()
	if err != nil {
		return "", time.Time{}, fmt.Errorf("生成分享令牌失败: %w", err)
	}
	expiresAt := time.Now().Add(defaultExpiry)
	if expiresInDays > 0 {
		expiresAt = time.Now().Add(time.Duration(expiresInDays) * 24 * time.Hour)
	}
	if _, err := database.Queries(ctx, s.Database).CreateShareToken(ctx, db.CreateShareTokenParams{
		DeliveryID: deliveryID,
		ExpiresAt:  expiresAt,
		TokenHash:  hashToken(raw),
	}); err != nil {
		return "", time.Time{}, fmt.Errorf("保存分享令牌失败: %w", err)
	}
	return raw, expiresAt, nil
}

// GetDeliveryByToken 通过明文令牌定位交付版本，链接失效时返回未找到。
func (s *Service) GetDeliveryByToken(ctx context.Context, raw string) (db.Delivery, error) {
	token, err := database.Queries(ctx, s.Database).GetShareTokenByHash(ctx, hashToken(raw))
	if err != nil {
		return db.Delivery{}, errs.NotFound("分享链接无效或已过期", err)
	}
	return database.Queries(ctx, s.Database).GetDelivery(ctx, token.DeliveryID)
}

// ListObjections 列出交付版本的全部异议及其回复。
func (s *Service) ListObjections(ctx context.Context, deliveryID uuid.UUID) ([]views.ObjectionView, error) {
	queries := database.Queries(ctx, s.Database)
	rows, err := queries.ListObjectionsByDelivery(ctx, deliveryID)
	if err != nil {
		return nil, fmt.Errorf("查询异议失败: %w", err)
	}
	result := make([]views.ObjectionView, 0, len(rows))
	for _, row := range rows {
		replies, err := queries.ListObjectionReplies(ctx, row.ID)
		if err != nil {
			return nil, fmt.Errorf("查询异议回复失败: %w", err)
		}
		result = append(result, views.FromObjection(row, replies))
	}
	return result, nil
}

// ListVerdicts 列出交付版本的全部验收裁定。
func (s *Service) ListVerdicts(ctx context.Context, deliveryID uuid.UUID) ([]views.AcceptanceVerdictView, error) {
	rows, err := database.Queries(ctx, s.Database).ListAcceptanceVerdicts(ctx, deliveryID)
	if err != nil {
		return nil, fmt.Errorf("查询验收裁定失败: %w", err)
	}
	result := make([]views.AcceptanceVerdictView, 0, len(rows))
	for _, row := range rows {
		result = append(result, views.FromAcceptanceVerdict(row))
	}
	return result, nil
}

// ReplyToObjection 开发者对客户异议作出回复。
func (s *Service) ReplyToObjection(ctx context.Context, objectionID, authorID uuid.UUID, content string) (db.ObjectionReply, error) {
	reply, err := database.Queries(ctx, s.Database).CreateObjectionReply(ctx, db.CreateObjectionReplyParams{
		AuthorID:    authorID,
		Content:     content,
		ObjectionID: objectionID,
	})
	if err != nil {
		return db.ObjectionReply{}, fmt.Errorf("回复异议失败: %w", err)
	}
	return reply, nil
}

// ResolveObjection 关闭异议，标记客户已复核。
func (s *Service) ResolveObjection(ctx context.Context, objectionID uuid.UUID) (db.Objection, error) {
	objection, err := database.Queries(ctx, s.Database).ResolveObjection(ctx, objectionID)
	if err != nil {
		return db.Objection{}, errs.BadRequest("该异议不存在或已关闭", err)
	}
	return objection, nil
}

// SaveVerdict 保存客户对单条验收条件的裁定结论。
func (s *Service) SaveVerdict(ctx context.Context, deliveryID, customerID uuid.UUID, criterionKey, verdict, comment string) (db.AcceptanceVerdict, error) {
	manifest := s.manifest(ctx, deliveryID)
	criterion, found := findCriterion(manifest.Criteria, criterionKey)
	if !found {
		return db.AcceptanceVerdict{}, errs.BadRequest("验收条件不存在于交付清单", nil)
	}
	return database.Queries(ctx, s.Database).UpsertAcceptanceVerdict(ctx, db.UpsertAcceptanceVerdictParams{
		Comment:          dbutil.TextFromString(comment),
		CriterionKey:     criterionKey,
		DecidedByID:      customerID,
		DeliveryID:       deliveryID,
		RequirementCode:  criterion.RequirementCode,
		RequirementTitle: findRequirementTitle(manifest, criterion.RequirementCode),
		Verdict:          verdict,
	})
}

// SubmitDecision 记录客户的最终确认结论，并按需登记异议。
func (s *Service) SubmitDecision(ctx context.Context, deliveryID uuid.UUID, req SubmitDecisionRequest) (db.Delivery, error) {
	delivery, err := database.Queries(ctx, s.Database).GetDelivery(ctx, deliveryID)
	if err != nil {
		return db.Delivery{}, errs.NotFound("交付版本不存在", err)
	}
	customer, err := s.EnsureCustomer(ctx, delivery.ProjectID, req.Email)
	if err != nil {
		return db.Delivery{}, err
	}
	if err := s.recordApproval(ctx, deliveryID, customer.ID, req.Decision, req.Comment); err != nil {
		return db.Delivery{}, err
	}
	if err := s.recordObjections(ctx, deliveryID, customer.ID, req.Objections); err != nil {
		return db.Delivery{}, err
	}
	status := statusApproved
	if req.Decision == decisionReject {
		status = statusRejected
	}
	return database.Queries(ctx, s.Database).UpdateDeliveryStatus(ctx, db.UpdateDeliveryStatusParams{
		ID:     deliveryID,
		Status: status,
	})
}

// EnsureCustomer 确认或创建客户账号，使验收结论可归因到具体客户。
func (s *Service) EnsureCustomer(ctx context.Context, projectID uuid.UUID, email string) (db.User, error) {
	queries := database.Queries(ctx, s.Database)
	project, err := queries.GetProject(ctx, projectID)
	if err != nil {
		return db.User{}, errs.NotFound("项目不存在", err)
	}
	normalized := strings.ToLower(strings.TrimSpace(email))
	user, err := queries.GetUserByEmail(ctx, normalized)
	if err == nil {
		// 邮箱已存在时必须校验组织归属，避免冒用其他组织客户的身份。
		if user.OrganizationID != project.OrganizationID {
			return db.User{}, errs.Forbidden("该邮箱已属于其他组织，无法用于本项目的验收", nil)
		}
		return user, nil
	}
	randomSecret, err := generateToken()
	if err != nil {
		return db.User{}, fmt.Errorf("生成客户初始凭证失败: %w", err)
	}
	passwordHash, err := auth.HashPassword(randomSecret)
	if err != nil {
		return db.User{}, fmt.Errorf("哈希客户初始凭证失败: %w", err)
	}
	return queries.CreateUser(ctx, db.CreateUserParams{
		DisplayName:    email,
		Email:          email,
		OrganizationID: project.OrganizationID,
		PasswordHash:   passwordHash,
		Role:           roleCustomer,
	})
}

// manifest 读取交付清单快照，供裁定校验与标题回填使用。
func (s *Service) manifest(ctx context.Context, deliveryID uuid.UUID) views.ManifestView {
	delivery, err := database.Queries(ctx, s.Database).GetDelivery(ctx, deliveryID)
	if err != nil {
		return views.ManifestView{}
	}
	return views.FromManifest(delivery.ManifestJson)
}

// findCriterion 在交付清单中查找验收条件。
func findCriterion(criteria []views.ManifestCriterionView, key string) (views.ManifestCriterionView, bool) {
	for _, criterion := range criteria {
		if criterion.Key == key {
			return criterion, true
		}
	}
	return views.ManifestCriterionView{}, false
}

// findRequirementTitle 按需求编号回填需求标题，便于客户在裁定列表中识别上下文。
func findRequirementTitle(manifest views.ManifestView, code string) string {
	for _, requirement := range manifest.Requirements {
		if requirement.Code == code {
			return requirement.Title
		}
	}
	return ""
}

// recordApproval 写入或更新客户对交付的确认结论。
func (s *Service) recordApproval(ctx context.Context, deliveryID, customerID uuid.UUID, decision, comment string) error {
	queries := database.Queries(ctx, s.Database)
	_, err := queries.GetApproval(ctx, db.GetApprovalParams{
		CustomerID: customerID,
		DeliveryID: deliveryID,
	})
	if err == nil {
		if _, err := queries.UpdateApproval(ctx, db.UpdateApprovalParams{
			Comment:    dbutil.TextFromString(comment),
			Decision:   decision,
			DeliveryID: deliveryID,
			CustomerID: customerID,
		}); err != nil {
			return fmt.Errorf("更新客户确认失败: %w", err)
		}
		return nil
	}
	if _, err := queries.CreateApproval(ctx, db.CreateApprovalParams{
		Comment:    dbutil.TextFromString(comment),
		CustomerID: customerID,
		Decision:   decision,
		DeliveryID: deliveryID,
	}); err != nil {
		return fmt.Errorf("保存客户确认失败: %w", err)
	}
	return nil
}

// recordObjections 登记客户在最终结论中提出的异议。
func (s *Service) recordObjections(ctx context.Context, deliveryID, customerID uuid.UUID, objections []ObjectionInput) error {
	queries := database.Queries(ctx, s.Database)
	for _, objection := range objections {
		if _, err := queries.CreateObjection(ctx, db.CreateObjectionParams{
			Content:     objection.Comment,
			CriterionID: objection.CriterionKey,
			DeliveryID:  deliveryID,
			RaisedByID:  customerID,
		}); err != nil {
			return fmt.Errorf("登记客户异议失败: %w", err)
		}
	}
	return nil
}

// generateToken 生成密码学安全的随机令牌。
func generateToken() (string, error) {
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

// hashToken 计算令牌摘要，使数据库泄露时无法反推有效链接。
func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// EnsureVerifiedCustomer 解析已通过邮箱验证的客户身份。
//
// 未验证的邮箱一律拒绝：交付证据的身份归因不能只凭「填写了一个邮箱」。
func (s *Service) EnsureVerifiedCustomer(ctx context.Context, deliveryID uuid.UUID, email string) (db.User, error) {
	normalized := strings.ToLower(strings.TrimSpace(email))
	queries := database.Queries(ctx, s.Database)
	identity, err := queries.GetAcceptanceIdentity(ctx, db.GetAcceptanceIdentityParams{
		DeliveryID: deliveryID,
		Email:      normalized,
	})
	if err != nil {
		return db.User{}, errs.Forbidden("请先完成邮箱验证再提交验收结论", nil)
	}
	if !identity.Verified {
		return db.User{}, errs.Forbidden("邮箱尚未验证，请先获取并填写验证码", nil)
	}
	if identity.UserID.Valid {
		return queries.GetUserByID(ctx, uuid.UUID(identity.UserID.Bytes))
	}
	return s.EnsureCustomer(ctx, identity.DeliveryID, normalized)
}

// RegisterVerifiedIdentity 登记客户在交付上的已验证身份。
func (s *Service) RegisterVerifiedIdentity(ctx context.Context, deliveryID uuid.UUID, email, code string) error {
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
	normalized := strings.ToLower(strings.TrimSpace(email))
	user, err := s.EnsureCustomer(ctx, deliveryProjectID(ctx, queries, deliveryID), normalized)
	if err != nil {
		return err
	}
	if _, err := queries.MarkUserEmailVerified(ctx, user.ID); err != nil {
		return errs.Internal(err)
	}
	_, err = queries.UpsertAcceptanceIdentity(ctx, db.UpsertAcceptanceIdentityParams{
		DeliveryID: deliveryID,
		Email:      normalized,
		UserID:     dbutil.NullableUUID(user.ID),
		Verified:   true,
	})
	if err != nil {
		return errs.Internal(err)
	}
	return nil
}

// deliveryProjectID 读取交付所属项目，用于校验客户邮箱的组织归属。
func deliveryProjectID(ctx context.Context, queries *db.Queries, deliveryID uuid.UUID) uuid.UUID {
	delivery, err := queries.GetDelivery(ctx, deliveryID)
	if err != nil {
		return uuid.Nil
	}
	return delivery.ProjectID
}
