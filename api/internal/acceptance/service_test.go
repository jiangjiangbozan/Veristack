// Package acceptance 的业务逻辑单元测试，不依赖数据库与网络。
package acceptance

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
	projectEvents []notification.ProjectEvent
	userEvents    []notification.ProjectEvent
}

// PublishProjectEvent 记录项目事件，供断言通知内容。
func (p *recordingPublisher) PublishProjectEvent(_ context.Context, event notification.ProjectEvent) {
	p.projectEvents = append(p.projectEvents, event)
}

// PublishUserEvent 记录用户事件，供断言通知对象。
func (p *recordingPublisher) PublishUserEvent(_ context.Context, _ uuid.UUID, event notification.ProjectEvent) {
	p.userEvents = append(p.userEvents, event)
}

func TestFindCriterion_ReturnsMatchingCriterion(t *testing.T) {
	criteria := []views.ManifestCriterionView{
		{Key: "REQ-001-acceptance", Description: "响应时间小于 2 秒"},
		{Key: "REQ-002-acceptance", Description: "导出文件可打开"},
	}

	for _, test := range []struct {
		name      string
		wantFound bool
		wantText  string
	}{
		{name: "命中第一条", wantFound: true, wantText: "响应时间小于 2 秒"},
		{name: "不存在的条件", wantFound: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			key := "REQ-001-acceptance"
			if !test.wantFound {
				key = "REQ-999-acceptance"
			}
			criterion, found := findCriterion(criteria, key)
			if found != test.wantFound {
				t.Fatalf("查找结果应为 %v，实际为 %v", test.wantFound, found)
			}
			if found && criterion.Description != test.wantText {
				t.Fatalf("条件描述应为 %q，实际为 %q", test.wantText, criterion.Description)
			}
		})
	}
}

func TestFindRequirementTitle_ReturnsTitleOrEmpty(t *testing.T) {
	manifest := views.ManifestView{Requirements: []views.ManifestRequirementView{
		{Code: "REQ-001", Title: "登录"},
	}}

	if title := findRequirementTitle(manifest, "REQ-001"); title != "登录" {
		t.Fatalf("应返回需求标题，实际为 %q", title)
	}
	if title := findRequirementTitle(manifest, "REQ-404"); title != "" {
		t.Fatalf("不存在的需求标题应为空，实际为 %q", title)
	}
}

func TestNotifyDecision_BuildsApprovedAndRejectedMessages(t *testing.T) {
	delivery := db.Delivery{ID: uuid.New(), ProjectID: uuid.New(), Version: 3}
	customer := db.User{Email: "customer@example.com"}
	publisher := &recordingPublisher{}
	service := Service{Notifier: publisher}

	service.notifyDecision(context.Background(), delivery, customer, SubmitDecisionRequest{Decision: decisionApprove}, statusApproved)
	if len(publisher.projectEvents) != 1 {
		t.Fatalf("应发布一条项目通知，实际为 %d", len(publisher.projectEvents))
	}
	approved := publisher.projectEvents[0]
	if approved.Type != "acceptance.decision" || !approved.SendEmail {
		t.Fatalf("验收通知类型或邮件标记不正确: %+v", approved)
	}
	if !strings.Contains(approved.Body, "验收通过") {
		t.Fatalf("通过结论正文不正确: %q", approved.Body)
	}
	if approved.Link != "/deliveries/"+delivery.ID.String() {
		t.Fatalf("通知链接不正确: %q", approved.Link)
	}

	publisher.projectEvents = nil
	service.notifyDecision(context.Background(), delivery, customer, SubmitDecisionRequest{
		Decision:   decisionReject,
		Objections: []ObjectionInput{{Comment: "样式不符", CriterionKey: "REQ-001"}},
	}, statusRejected)
	if len(publisher.projectEvents) != 1 {
		t.Fatalf("应发布一条项目通知，实际为 %d", len(publisher.projectEvents))
	}
	rejected := publisher.projectEvents[0]
	if !strings.Contains(rejected.Body, "1 条异议") {
		t.Fatalf("驳回结论正文应包含异议数量: %q", rejected.Body)
	}
	if !strings.Contains(rejected.Title, "异议") {
		t.Fatalf("驳回结论标题不正确: %q", rejected.Title)
	}
}

func TestNotifyDecision_NilNotifierDoesNotPanic(t *testing.T) {
	service := Service{}
	service.notifyDecision(context.Background(), db.Delivery{}, db.User{}, SubmitDecisionRequest{}, statusApproved)
}

func TestGenerateVerificationCode_ReturnsSixDigits(t *testing.T) {
	for index := 0; index < 20; index++ {
		code, err := generateVerificationCode()
		if err != nil {
			t.Fatalf("生成验证码失败: %v", err)
		}
		if len(code) != verificationCodeLength {
			t.Fatalf("验证码长度应为 %d，实际为 %d", verificationCodeLength, len(code))
		}
		for _, digit := range code {
			if digit < '0' || digit > '9' {
				t.Fatalf("验证码应只包含数字，实际为 %q", code)
			}
		}
	}
}

func TestHashVerificationCode_NormalizesWhitespace(t *testing.T) {
	first := hashVerificationCode(" 123456 ")
	second := hashVerificationCode("123456")
	if first != second {
		t.Fatalf("去除首尾空白后的摘要应一致: %q != %q", first, second)
	}
	if first == hashVerificationCode("654321") {
		t.Fatalf("不同验证码的摘要不应相同")
	}
}
