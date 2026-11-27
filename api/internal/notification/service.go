// Package notification 提供站内通知与邮件通知，供验收、变更等业务流程调用。
package notification

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"api/internal/platform/database"
	"api/internal/platform/mailer"
	"api/internal/platform/outbox"
)

const (
	defaultListLimit = 20
	maxListLimit     = 100
)

// Notification 是一条站内通知记录。
type Notification struct {
	Body           string             // 通知正文
	CreatedAt      time.Time          // 创建时间
	ID             uuid.UUID          // 通知 ID
	Link           string             // 站内跳转链接
	OrganizationID uuid.UUID          // 所属组织
	ReadAt         pgtype.Timestamptz // 阅读时间，未读为空
	RecipientID    uuid.UUID          // 接收人
	Title          string             // 通知标题
	Type           string             // 事件类型
}

// notificationRecipient 是站内通知与邮件接收人。
type notificationRecipient struct {
	DisplayName string    // 用户显示名
	Email       string    // 邮箱
	UserID      uuid.UUID // 用户 ID
}

// ProjectEvent 描述一次需要通知项目相关成员的业务事件。
type ProjectEvent struct {
	Body      string    // 通知正文
	DedupeKey string    // 幂等键，为空表示不去重
	Link      string    // 站内跳转链接
	ProjectID uuid.UUID // 所属项目
	SendEmail bool      // 是否同时发送邮件
	Title     string    // 通知标题
	Type      string    // 事件类型
}

// Publisher 抽象项目与用户通知发布能力，便于业务服务注入测试替身。
type Publisher interface {
	PublishProjectEvent(ctx context.Context, event ProjectEvent)
	PublishUserEvent(ctx context.Context, userID uuid.UUID, event ProjectEvent)
}

// Service 提供通知的写入、查询与已读操作。
type Service struct {
	Database      *pgxpool.Pool     // 数据库连接池
	EmailQueue    outbox.EmailQueue // 邮件任务队列，未配置时回退为同步发送
	Logger        *slog.Logger      // 日志记录器
	Mailer        mailer.Mailer     // 邮件发送通道，可为空
	PublicBaseURL string            // 前端公网地址，用于邮件中的跳转链接
}

// List 分页列出接收人的通知，按创建时间倒序。
func (s *Service) List(ctx context.Context, recipientID uuid.UUID, limit, offset int32) ([]Notification, error) {
	if limit <= 0 || limit > maxListLimit {
		limit = defaultListLimit
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := s.Database.Query(ctx, `
		SELECT id, organization_id, recipient_id, type, title, body, link, read_at, created_at
		FROM notifications
		WHERE recipient_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`, recipientID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("查询通知失败: %w", err)
	}
	defer rows.Close()
	result := make([]Notification, 0)
	for rows.Next() {
		var item Notification
		if err := rows.Scan(&item.ID, &item.OrganizationID, &item.RecipientID, &item.Type, &item.Title, &item.Body, &item.Link, &item.ReadAt, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("读取通知失败: %w", err)
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

// UnreadCount 返回接收人的未读通知数量。
func (s *Service) UnreadCount(ctx context.Context, recipientID uuid.UUID) (int64, error) {
	var count int64
	err := s.Database.QueryRow(ctx, `
		SELECT count(*) FROM notifications
		WHERE recipient_id = $1 AND read_at IS NULL`, recipientID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("统计未读通知失败: %w", err)
	}
	return count, nil
}

// MarkRead 把一条通知标记为已读；重复标记不会报错。
func (s *Service) MarkRead(ctx context.Context, recipientID, notificationID uuid.UUID) error {
	exec := database.Executor(ctx, s.Database)
	_, err := exec.Exec(ctx, `
		UPDATE notifications SET read_at = now()
		WHERE id = $1 AND recipient_id = $2 AND read_at IS NULL`, notificationID, recipientID)
	if err != nil {
		return fmt.Errorf("标记通知已读失败: %w", err)
	}
	return nil
}

// MarkAllRead 把接收人的全部未读通知标记为已读，返回本次更新数量。
func (s *Service) MarkAllRead(ctx context.Context, recipientID uuid.UUID) (int64, error) {
	exec := database.Executor(ctx, s.Database)
	tag, err := exec.Exec(ctx, `
		UPDATE notifications SET read_at = now()
		WHERE recipient_id = $1 AND read_at IS NULL`, recipientID)
	if err != nil {
		return 0, fmt.Errorf("标记全部通知已读失败: %w", err)
	}
	return tag.RowsAffected(), nil
}

// PublishProjectEvent 向项目相关成员（组织 owner/admin 与项目成员）发布通知。
//
// 站内通知与邮件任务必须同事务落库：入队失败会让整个写请求回滚，
// 避免业务已提交但邮件任务丢失；真正的 SMTP 失败则由 Worker 重试。
func (s *Service) PublishProjectEvent(ctx context.Context, event ProjectEvent) {
	exec := database.Executor(ctx, s.Database)
	var organizationID uuid.UUID
	err := exec.QueryRow(ctx, `
		SELECT organization_id FROM projects
		WHERE id = $1 AND deleted_at IS NULL`, event.ProjectID).Scan(&organizationID)
	if err != nil {
		s.logError("读取项目所属组织失败", err)
		return
	}
	recipients := s.projectRecipients(ctx, organizationID, event.ProjectID)
	for _, recipient := range recipients {
		if err := s.create(ctx, organizationID, recipient.UserID, event, recipient.Email, recipient.DisplayName); err != nil {
			s.logError("写入通知失败", err)
		}
	}
}

// projectRecipients 读取项目通知接收人，先读完再写通知。
//
// pgx 的 Rows 占用连接，边读边写同一事务会触发连接忙。
func (s *Service) projectRecipients(ctx context.Context, organizationID, projectID uuid.UUID) []notificationRecipient {
	exec := database.Executor(ctx, s.Database)
	rows, err := exec.Query(ctx, `
		SELECT DISTINCT u.id, u.email, u.display_name
		FROM users u
		WHERE u.organization_id = $1 AND u.status = 'active' AND u.deleted_at IS NULL
		  AND (u.role IN ('owner', 'admin')
		       OR EXISTS (SELECT 1 FROM project_members pm WHERE pm.project_id = $2 AND pm.user_id = u.id))`,
		organizationID, projectID)
	if err != nil {
		s.logError("查询通知接收人失败", err)
		return nil
	}
	defer rows.Close()
	recipients := make([]notificationRecipient, 0)
	for rows.Next() {
		var recipient notificationRecipient
		if err := rows.Scan(&recipient.UserID, &recipient.Email, &recipient.DisplayName); err != nil {
			s.logError("读取通知接收人失败", err)
			continue
		}
		recipients = append(recipients, recipient)
	}
	if err := rows.Err(); err != nil {
		s.logError("遍历通知接收人失败", err)
	}
	return recipients
}

// PublishUserEvent 向指定用户发布通知，用于回复客户异议等定向事件。
func (s *Service) PublishUserEvent(ctx context.Context, userID uuid.UUID, event ProjectEvent) {
	exec := database.Executor(ctx, s.Database)
	var organizationID uuid.UUID
	var email, displayName string
	err := exec.QueryRow(ctx, `
		SELECT organization_id, email, display_name FROM users
		WHERE id = $1 AND status = 'active' AND deleted_at IS NULL`, userID).Scan(&organizationID, &email, &displayName)
	if err != nil {
		if err != pgx.ErrNoRows {
			s.logError("读取通知接收人失败", err)
		}
		return
	}
	if err := s.create(ctx, organizationID, userID, event, email, displayName); err != nil {
		s.logError("写入通知失败", err)
	}
}

// create 写入一条通知，并按需发送邮件。
func (s *Service) create(ctx context.Context, organizationID, recipientID uuid.UUID, event ProjectEvent, email, displayName string) error {
	exec := database.Executor(ctx, s.Database)
	var id uuid.UUID
	err := exec.QueryRow(ctx, `
		INSERT INTO notifications (organization_id, recipient_id, type, title, body, link, dedupe_key)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT DO NOTHING
		RETURNING id`,
		organizationID, recipientID, event.Type, event.Title, event.Body, event.Link, event.DedupeKey).Scan(&id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil
		}
		return err
	}
	if !event.SendEmail || email == "" {
		return nil
	}
	body := buildEmailBody(displayName, event.Body, s.absoluteLink(event.Link))
	if s.EmailQueue != nil {
		// 通知与邮件任务在同一事务内落库，避免业务回滚后邮件已经发出。
		return s.EmailQueue.EnqueueEmail(ctx, outbox.EmailPayload{
			Body:           body,
			NotificationID: id.String(),
			Subject:        event.Title,
			To:             email,
		}, "notification:"+id.String())
	}
	if s.Mailer == nil {
		return nil
	}
	if err := s.Mailer.Send(ctx, email, event.Title, body); err != nil {
		s.logError("通知邮件发送失败", err)
		return nil
	}
	_, _ = exec.Exec(ctx, `UPDATE notifications SET emailed_at = now() WHERE id = $1`, id)
	return nil
}

// buildEmailBody 组合邮件正文，附加可选站内链接。
func buildEmailBody(displayName, body, link string) string {
	greeting := "你好"
	if displayName != "" {
		greeting = fmt.Sprintf("你好，%s", displayName)
	}
	result := fmt.Sprintf("%s：\n\n%s", greeting, body)
	if link != "" {
		result += fmt.Sprintf("\n\n查看详情：%s", link)
	}
	return result
}

// absoluteLink 把站内相对路径补全为公网地址，避免邮件里的链接无法点击。
func (s *Service) absoluteLink(link string) string {
	if link == "" || !strings.HasPrefix(link, "/") {
		return link
	}
	base := strings.TrimRight(s.PublicBaseURL, "/")
	if base == "" {
		base = "http://localhost:5173"
	}
	return base + link
}

// logError 输出通知链路错误，避免影响主业务返回。
func (s *Service) logError(message string, err error) {
	if s.Logger == nil {
		return
	}
	s.Logger.Error(message, "error", err)
}
