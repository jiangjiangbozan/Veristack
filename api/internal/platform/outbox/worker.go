// Package outbox 的邮件 Worker，负责领取任务、发送邮件并记录重试状态。
package outbox

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"api/internal/platform/database"
	"api/internal/platform/db"
	"api/internal/platform/mailer"
)

const (
	defaultEmailBatchSize    = 10
	defaultEmailMaxAttempts  = 5
	defaultEmailPollInterval = 2 * time.Second
	defaultEmailRetryDelay   = 60 * time.Second
	emailTaskErrorMaxLength  = 500
)

// EmailWorker 轮询邮件任务并调用邮件通道发送。
type EmailWorker struct {
	BatchSize    int32         // 每批领取数量，未配置时使用默认值
	Database     *pgxpool.Pool // 数据库连接池
	Logger       *slog.Logger  // 结构化日志器
	Mailer       mailer.Mailer // 邮件发送通道
	MaxAttempts  int32         // 最大尝试次数，未配置时使用默认值
	PollInterval time.Duration // 轮询间隔，未配置时使用默认值
	RetryDelay   time.Duration // 失败后的重试间隔，未配置时使用默认值
}

// Run 持续轮询邮件任务，直到上下文结束。
func (w *EmailWorker) Run(ctx context.Context) {
	interval := w.PollInterval
	if interval <= 0 {
		interval = defaultEmailPollInterval
	}
	w.processBatch(ctx)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.processBatch(ctx)
		}
	}
}

// processBatch 领取并处理一批邮件任务，单条失败不影响其余任务。
func (w *EmailWorker) processBatch(ctx context.Context) {
	batchSize := w.BatchSize
	if batchSize <= 0 {
		batchSize = defaultEmailBatchSize
	}
	tasks, err := db.New(w.Database).ClaimPendingEmailOutboxTasks(ctx, batchSize)
	if err != nil {
		w.logError("领取邮件任务失败", err)
		return
	}
	for _, task := range tasks {
		w.processTask(ctx, task)
	}
}

// processTask 处理单条任务，成功或失败都写回任务状态。
func (w *EmailWorker) processTask(ctx context.Context, task db.OutboxTask) {
	if err := w.handleTask(ctx, task); err != nil {
		w.logError("发送邮件失败", err)
		w.failTask(ctx, task, err)
		return
	}
	if _, err := db.New(w.Database).CompleteOutboxTask(ctx, task.ID); err != nil {
		w.logError("标记邮件任务完成失败", err)
	}
}

// handleTask 解析载荷并发送邮件；通知回填失败不回滚已发送的邮件。
func (w *EmailWorker) handleTask(ctx context.Context, task db.OutboxTask) error {
	payload, err := decodeEmailPayload(task.Payload)
	if err != nil {
		return err
	}
	if w.Mailer == nil {
		return fmt.Errorf("邮件通道未配置")
	}
	if err := w.Mailer.Send(ctx, payload.To, payload.Subject, payload.Body); err != nil {
		return fmt.Errorf("发送邮件失败: %w", err)
	}
	if payload.NotificationID != "" {
		if _, err := database.Executor(ctx, w.Database).Exec(ctx, `
			UPDATE notifications SET emailed_at = now()
			WHERE id = $1 AND emailed_at IS NULL`, payload.NotificationID); err != nil {
			w.logError("回填通知邮件发送时间失败", err)
		}
	}
	return nil
}

// failTask 按尝试次数决定继续重试还是标记为最终失败。
func (w *EmailWorker) failTask(ctx context.Context, task db.OutboxTask, cause error) {
	maxAttempts := w.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = defaultEmailMaxAttempts
	}
	status := emailTaskStatus(task.Attempts, maxAttempts)
	retryDelay := w.RetryDelay
	if retryDelay <= 0 {
		retryDelay = defaultEmailRetryDelay
	}
	lastError := truncateEmailError(cause.Error())
	if _, err := db.New(w.Database).FailOutboxTask(ctx, db.FailOutboxTaskParams{
		ID:        task.ID,
		LastError: pgtype.Text{String: lastError, Valid: true},
		Secs:      retryDelay.Seconds(),
		Status:    status,
	}); err != nil {
		w.logError("标记邮件任务失败状态出错", err)
	}
}

// logError 输出任务链路错误，避免单条失败影响 Worker 循环。
func (w *EmailWorker) logError(message string, err error) {
	if w.Logger == nil {
		return
	}
	w.Logger.Error(message, "error", err)
}

// emailTaskStatus 返回本次失败后应写入的任务状态，供单元测试直接验证。
func emailTaskStatus(attempts, maxAttempts int32) string {
	if attempts >= maxAttempts {
		return "failed"
	}
	return "pending"
}

// truncateEmailError 截断过长的错误文本，避免撑大任务表。
func truncateEmailError(message string) string {
	message = strings.TrimSpace(message)
	if len(message) > emailTaskErrorMaxLength {
		return message[:emailTaskErrorMaxLength]
	}
	return message
}
