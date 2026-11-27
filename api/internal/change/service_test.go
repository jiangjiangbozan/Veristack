// Package change 的业务逻辑单元测试，不依赖数据库与网络。
package change

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"

	"api/internal/notification"
	"api/internal/platform/db"
	"api/internal/platform/views"
)

type recordingPublisher struct {
	events []notification.ProjectEvent
}

// PublishProjectEvent 记录项目事件。
func (p *recordingPublisher) PublishProjectEvent(_ context.Context, event notification.ProjectEvent) {
	p.events = append(p.events, event)
}

// PublishUserEvent 记录用户事件。
func (p *recordingPublisher) PublishUserEvent(_ context.Context, _ uuid.UUID, event notification.ProjectEvent) {
	p.events = append(p.events, event)
}

func TestApplyChanges_ModifyReplacesExistingRequirement(t *testing.T) {
	existingID := uuid.New()
	current := []views.RequirementView{{
		Code: "REQ-001", ID: existingID, Title: "旧标题", Status: "approved",
	}}
	changes := []RequirementChange{{
		Code: "REQ-001", Description: "新描述", Operation: "modify", Priority: "high", Title: "新标题",
	}}

	result := applyChanges(current, changes)

	if len(result) != 1 {
		t.Fatalf("修改不应产生重复需求，实际为 %d 条: %+v", len(result), result)
	}
	if result[0].ID != existingID {
		t.Fatalf("修改应保留原需求 ID，实际为 %s", result[0].ID)
	}
	if result[0].Title != "新标题" || result[0].Status != "pending" {
		t.Fatalf("修改后的需求内容不正确: %+v", result[0])
	}
}

func TestApplyChanges_AddAndRemove(t *testing.T) {
	firstID := uuid.New()
	current := []views.RequirementView{
		{Code: "REQ-001", ID: firstID, Title: "登录"},
		{Code: "REQ-002", ID: uuid.New(), Title: "导出"},
	}
	changes := []RequirementChange{
		{Code: "REQ-001", Operation: "remove"},
		{Code: "REQ-003", Operation: "add", Priority: "medium", Title: "审计"},
	}

	result := applyChanges(current, changes)

	if len(result) != 2 {
		t.Fatalf("删除一条、新增一条后应为 2 条，实际为 %d", len(result))
	}
	if result[0].Code != "REQ-002" || result[1].Code != "REQ-003" {
		t.Fatalf("变更后的需求顺序或编号不正确: %+v", result)
	}
}

func TestApplyChanges_ModifyMissingRequirementAppends(t *testing.T) {
	changes := []RequirementChange{{Code: "REQ-009", Operation: "modify", Title: "新增需求"}}

	result := applyChanges(nil, changes)

	if len(result) != 1 || result[0].Code != "REQ-009" {
		t.Fatalf("修改不存在的需求应追加一条，实际为 %+v", result)
	}
}

func TestParseRequirements_InvalidJSONReturnsEmpty(t *testing.T) {
	result := parseRequirements([]byte("{invalid"))

	if len(result) != 0 {
		t.Fatalf("非法 JSON 应返回空列表，实际为 %+v", result)
	}
}

func TestNotifyDecision_BuildsApprovedAndRejectedMessages(t *testing.T) {
	changeRequest := db.ChangeRequest{
		ID:          uuid.New(),
		ProjectID:   uuid.New(),
		RequesterID: uuid.New(),
	}
	publisher := &recordingPublisher{}
	service := Service{Notifier: publisher}

	service.notifyDecision(context.Background(), changeRequest, statusApproved)
	if len(publisher.events) != 1 {
		t.Fatalf("应发布一条通知，实际为 %d", len(publisher.events))
	}
	approved := publisher.events[0]
	if approved.Type != "change.decided" || !approved.SendEmail {
		t.Fatalf("变更通知类型或邮件标记不正确: %+v", approved)
	}
	if !strings.Contains(approved.Title, "批准") {
		t.Fatalf("批准通知标题不正确: %q", approved.Title)
	}
	if approved.Link != "/projects/"+changeRequest.ProjectID.String() {
		t.Fatalf("通知链接不正确: %q", approved.Link)
	}

	publisher.events = nil
	service.notifyDecision(context.Background(), changeRequest, statusRejected)
	if len(publisher.events) != 1 {
		t.Fatalf("应发布一条通知，实际为 %d", len(publisher.events))
	}
	rejected := publisher.events[0]
	if !strings.Contains(rejected.Title, "未通过") {
		t.Fatalf("驳回通知标题不正确: %q", rejected.Title)
	}
}

func TestNotifyDecision_NilNotifierDoesNotPanic(t *testing.T) {
	service := Service{}
	service.notifyDecision(context.Background(), db.ChangeRequest{}, statusRejected)
}
