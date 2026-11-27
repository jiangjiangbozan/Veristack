// Package outbox 提供基于数据库任务表的异步副作用投递，当前承载邮件发送。
package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"api/internal/platform/database"
	"api/internal/platform/db"
)

const EmailEventType = "email.send"

// EmailPayload 是一条邮件投递任务的最小载荷。
type EmailPayload struct {
	Body           string `json:"body"`                     // 邮件正文
	NotificationID string `json:"notificationId,omitempty"` // 关联站内通知，发送成功后回填 emailed_at
	Subject        string `json:"subject"`                  // 邮件主题
	To             string `json:"to"`                       // 收件人
}

// EmailQueue 抽象邮件入队能力，业务服务只负责把邮件写入当前事务。
type EmailQueue interface {
	EnqueueEmail(ctx context.Context, payload EmailPayload, dedupeKey string) error
}

// TaskCreator 抽象 outbox 任务写入，便于测试幂等分支。
type TaskCreator interface {
	CreateOutboxTask(ctx context.Context, arg db.CreateOutboxTaskParams) (db.OutboxTask, error)
}

// EmailWriter 把邮件任务写入 outbox_tasks，和业务写入共用同一个事务。
type EmailWriter struct {
	Creator  TaskCreator   // 测试替身；为空时使用当前上下文事务
	Database *pgxpool.Pool // 数据库连接池
}

// NewEmailWriter 创建邮件任务写入器。
func NewEmailWriter(pool *pgxpool.Pool) *EmailWriter {
	return &EmailWriter{Database: pool}
}

// EnqueueEmail 把邮件任务加入当前上下文的事务；重复的 dedupeKey 视为已入队。
func (w *EmailWriter) EnqueueEmail(ctx context.Context, payload EmailPayload, dedupeKey string) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化邮件任务失败: %w", err)
	}
	creator := w.Creator
	if creator == nil {
		creator = database.Queries(ctx, w.Database)
	}
	_, err = creator.CreateOutboxTask(ctx, db.CreateOutboxTaskParams{
		DedupeKey: dedupeKey,
		EventType: EmailEventType,
		Payload:   body,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("写入邮件任务失败: %w", err)
	}
	return nil
}

// decodeEmailPayload 解析邮件任务载荷。
func decodeEmailPayload(raw []byte) (EmailPayload, error) {
	var payload EmailPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return EmailPayload{}, fmt.Errorf("解析邮件任务失败: %w", err)
	}
	if payload.To == "" {
		return EmailPayload{}, fmt.Errorf("邮件任务缺少收件人")
	}
	return payload, nil
}
