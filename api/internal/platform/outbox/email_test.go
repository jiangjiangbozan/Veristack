// Package outbox 的邮件任务单元测试，不依赖数据库与网络。
package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"

	"api/internal/platform/db"
	"api/internal/platform/mailer"
)

type recordingCreator struct {
	params db.CreateOutboxTaskParams
	result db.OutboxTask
	err    error
}

// CreateOutboxTask 记录任务参数并返回预置结果。
func (c *recordingCreator) CreateOutboxTask(_ context.Context, arg db.CreateOutboxTaskParams) (db.OutboxTask, error) {
	c.params = arg
	return c.result, c.err
}

type recordingMailer struct {
	body    string
	err     error
	subject string
	to      string
}

// Send 记录邮件发送参数。
func (m *recordingMailer) Send(_ context.Context, to, subject, body string) error {
	m.to = to
	m.subject = subject
	m.body = body
	return m.err
}

func TestEnqueueEmail_WritesIdempotentTask(t *testing.T) {
	creator := &recordingCreator{}
	writer := &EmailWriter{Creator: creator}
	payload := EmailPayload{Body: "正文", Subject: "主题", To: "user@example.com"}

	if err := writer.EnqueueEmail(context.Background(), payload, "notification:1"); err != nil {
		t.Fatalf("入队失败: %v", err)
	}
	if creator.params.EventType != EmailEventType {
		t.Fatalf("事件类型应为 %s，实际为 %s", EmailEventType, creator.params.EventType)
	}
	if creator.params.DedupeKey != "notification:1" {
		t.Fatalf("去重键不正确: %q", creator.params.DedupeKey)
	}
	var decoded EmailPayload
	if err := json.Unmarshal(creator.params.Payload, &decoded); err != nil {
		t.Fatalf("任务载荷不是合法 JSON: %v", err)
	}
	if decoded.To != payload.To || decoded.Subject != payload.Subject || decoded.Body != payload.Body {
		t.Fatalf("任务载荷内容不正确: %+v", decoded)
	}
}

func TestEnqueueEmail_TreatsDuplicateAsSuccess(t *testing.T) {
	writer := &EmailWriter{Creator: &recordingCreator{err: pgx.ErrNoRows}}

	if err := writer.EnqueueEmail(context.Background(), EmailPayload{To: "user@example.com"}, "same"); err != nil {
		t.Fatalf("重复任务应视为已入队: %v", err)
	}
}

func TestEnqueueEmail_ReturnsCreatorError(t *testing.T) {
	writer := &EmailWriter{Creator: &recordingCreator{err: errors.New("db down")}}

	if err := writer.EnqueueEmail(context.Background(), EmailPayload{To: "user@example.com"}, "key"); err == nil {
		t.Fatalf("数据库错误应向上返回")
	}
}

func TestDecodeEmailPayload_ValidatesRecipient(t *testing.T) {
	valid := []byte(`{"to":"user@example.com","subject":"主题","body":"正文"}`)
	payload, err := decodeEmailPayload(valid)
	if err != nil || payload.To != "user@example.com" {
		t.Fatalf("合法载荷解析失败: (%+v, %v)", payload, err)
	}

	if _, err := decodeEmailPayload([]byte(`{"subject":"主题"}`)); err == nil {
		t.Fatalf("缺少收件人应报错")
	}
	if _, err := decodeEmailPayload([]byte(`{invalid`)); err == nil {
		t.Fatalf("非法 JSON 应报错")
	}
}

func TestEmailWorker_HandleTaskSendsMail(t *testing.T) {
	m := &recordingMailer{}
	worker := EmailWorker{Mailer: m}
	task := db.OutboxTask{Payload: []byte(`{"to":"user@example.com","subject":"主题","body":"正文"}`)}

	if err := worker.handleTask(context.Background(), task); err != nil {
		t.Fatalf("处理邮件任务失败: %v", err)
	}
	if m.to != "user@example.com" || m.subject != "主题" || m.body != "正文" {
		t.Fatalf("邮件发送参数不正确: %+v", m)
	}
}

func TestEmailWorker_HandleTaskReturnsMailerError(t *testing.T) {
	worker := EmailWorker{Mailer: &recordingMailer{err: errors.New("smtp down")}}
	task := db.OutboxTask{Payload: []byte(`{"to":"user@example.com"}`)}

	if err := worker.handleTask(context.Background(), task); err == nil {
		t.Fatalf("邮件通道错误应向上返回以便重试")
	}
}

func TestEmailWorker_HandleTaskRejectsMissingMailer(t *testing.T) {
	worker := EmailWorker{}
	task := db.OutboxTask{Payload: []byte(`{"to":"user@example.com"}`)}

	if err := worker.handleTask(context.Background(), task); err == nil {
		t.Fatalf("未配置邮件通道时应失败")
	}
}

func TestEmailTaskStatus(t *testing.T) {
	for _, test := range []struct {
		name        string
		attempts    int32
		maxAttempts int32
		want        string
	}{
		{name: "未达上限继续重试", attempts: 2, maxAttempts: 5, want: "pending"},
		{name: "达到上限最终失败", attempts: 5, maxAttempts: 5, want: "failed"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := emailTaskStatus(test.attempts, test.maxAttempts); got != test.want {
				t.Fatalf("状态应为 %s，实际为 %s", test.want, got)
			}
		})
	}
}

func TestTruncateEmailError(t *testing.T) {
	message := make([]byte, emailTaskErrorMaxLength+10)
	for index := range message {
		message[index] = 'x'
	}
	result := truncateEmailError(string(message))
	if len(result) != emailTaskErrorMaxLength {
		t.Fatalf("错误文本应截断到 %d 字节，实际为 %d", emailTaskErrorMaxLength, len(result))
	}
}

var _ mailer.Mailer = (*recordingMailer)(nil)
