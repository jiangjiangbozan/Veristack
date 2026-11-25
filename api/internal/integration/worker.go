// Package integration 的后台任务处理，负责消费 Webhook 事件并把提交关联到需求。
package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"api/internal/platform/datetime"
	"api/internal/platform/db"
	"api/internal/platform/dbutil"
)

const (
	workerPollInterval = 5 * time.Second
	workerBatchSize    = 20
	maxTaskAttempts    = 5
	maxNoteLength      = 2000
	retryDelaySeconds  = 60
	shortHashLength    = 7
)

// emptyUUID 表示未关联到具体对象。
var emptyUUID = uuid.Nil

// requirementCodePattern 匹配提交信息中的需求编号，用于把提交关联到需求。
var requirementCodePattern = regexp.MustCompile(`(?i)REQ-\d{3}`)

// Worker 消费 outbox 任务，把外部事件转换为交付证据。
type Worker struct {
	Database *pgxpool.Pool // 数据库连接池
	Logger   *slog.Logger  // 结构化日志器
}

// Run 持续轮询任务队列，直到上下文结束。
func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(workerPollInterval)
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

// processBatch 处理一批到期任务，单个任务失败不影响其余任务。
func (w *Worker) processBatch(ctx context.Context) {
	tasks, err := db.New(w.Database).ClaimPendingOutboxTasks(ctx, workerBatchSize)
	if err != nil {
		w.Logger.Error("领取异步任务失败", "error", err)
		return
	}
	for _, task := range tasks {
		w.processTask(ctx, task)
	}
}

// processTask 处理单个任务，失败时按次数安排重试。
func (w *Worker) processTask(ctx context.Context, task db.OutboxTask) {
	if err := w.handleTask(ctx, task); err != nil {
		w.Logger.Error("处理异步任务失败", "eventType", task.EventType, "error", err)
		status := "pending"
		if task.Attempts >= maxTaskAttempts {
			status = "failed"
		}
		if _, updateErr := db.New(w.Database).FailOutboxTask(ctx, db.FailOutboxTaskParams{
			ID:     task.ID,
			Secs:   retryDelaySeconds,
			Status: status,
		}); updateErr != nil {
			w.Logger.Error("标记任务失败状态出错", "error", updateErr)
		}
		return
	}
	if _, err := db.New(w.Database).CompleteOutboxTask(ctx, task.ID); err != nil {
		w.Logger.Error("标记任务完成状态出错", "error", err)
	}
}

// handleTask 按事件类型分发处理逻辑。
func (w *Worker) handleTask(ctx context.Context, task db.OutboxTask) error {
	switch task.EventType {
	case "push", "Push Hook":
		return w.handlePush(ctx, task)
	default:
		return nil
	}
}

// handlePush 从 push 事件中提取提交信息，关联需求并登记为证据。
func (w *Worker) handlePush(ctx context.Context, task db.OutboxTask) error {
	projectID := dbutil.OptionalUUID(task.ProjectID)
	if projectID == emptyUUID {
		return nil
	}
	payload, err := decodePushPayload(task.Payload)
	if err != nil {
		return err
	}
	for _, commit := range payload.Commits {
		if err := w.recordCommitEvidence(ctx, projectID, commit); err != nil {
			return err
		}
	}
	return nil
}

// recordCommitEvidence 把一次提交登记为 Git 提交类型的证据。
func (w *Worker) recordCommitEvidence(ctx context.Context, projectID uuid.UUID, commit pushCommit) error {
	requirementID, err := w.resolveRequirement(ctx, projectID, commit.Message)
	if err != nil {
		return err
	}
	note := commit.Message
	if len(note) > maxNoteLength {
		note = note[:maxNoteLength]
	}
	params := db.CreateEvidenceItemParams{
		DeliveryID:    dbutil.NullableUUID(emptyUUID),
		Note:          dbutil.TextFromString(note),
		OccurredAt:    datetime.Now(),
		ProjectID:     projectID,
		RequirementID: dbutil.NullableUUID(requirementID),
		Source:        dbutil.TextFromString(commit.Author),
		Title:         fmt.Sprintf("提交 %s", shortHash(commit.Hash)),
		Type:          "git_commit",
	}
	if _, err := db.New(w.Database).CreateEvidenceItem(ctx, params); err != nil {
		return fmt.Errorf("登记提交证据失败: %w", err)
	}
	return nil
}

// resolveRequirement 按提交信息中的需求编号定位需求。
func (w *Worker) resolveRequirement(ctx context.Context, projectID uuid.UUID, message string) (uuid.UUID, error) {
	codes := requirementCodePattern.FindAllString(message, -1)
	if len(codes) == 0 {
		return emptyUUID, nil
	}
	requirement, err := db.New(w.Database).GetRequirementByCode(ctx, db.GetRequirementByCodeParams{
		Code:      codes[0],
		ProjectID: projectID,
	})
	if err != nil {
		return emptyUUID, nil
	}
	return requirement.ID, nil
}

// pushPayload 是 GitHub push 事件中我们关心的字段。
type pushPayload struct {
	Commits []pushCommit `json:"commits"`
}

// pushCommit 是一次提交的基本信息。
type pushCommit struct {
	Author  string `json:"author"`
	Hash    string `json:"id"`
	Message string `json:"message"`
}

// decodePushPayload 解析 push 事件载荷。
func decodePushPayload(raw []byte) (pushPayload, error) {
	var payload pushPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return pushPayload{}, fmt.Errorf("解析 push 事件失败: %w", err)
	}
	return payload, nil
}

// shortHash 截短提交哈希，便于在界面上展示。
func shortHash(hash string) string {
	if len(hash) <= shortHashLength {
		return hash
	}
	return hash[:shortHashLength]
}
