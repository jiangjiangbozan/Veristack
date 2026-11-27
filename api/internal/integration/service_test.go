// Package integration 的业务逻辑单元测试，不依赖数据库与网络。
package integration

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"api/internal/platform/config"
	"api/internal/platform/db"
)

func TestVerifySignature_GitHub(t *testing.T) {
	payload := []byte(`{"ref":"refs/heads/main"}`)
	service := Service{Webhook: config.WebhookConfig{GitHubSecret: "secret"}}
	signature := "sha256=" + hex.EncodeToString(hmacSHA256("secret", payload))

	for _, test := range []struct {
		name      string
		signature string
		wantError bool
	}{
		{name: "正确签名", signature: signature},
		{name: "空签名", signature: "", wantError: true},
		{name: "篡改签名", signature: "sha256=bad", wantError: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := service.VerifySignature(providerGitHub, test.signature, payload)
			if (err != nil) != test.wantError {
				t.Fatalf("错误结果应为 %v，实际为 %v", test.wantError, err)
			}
		})
	}
}

func TestVerifySignature_AllowsWhenSecretMissing(t *testing.T) {
	service := Service{}
	if err := service.VerifySignature(providerGitHub, "", []byte("payload")); err != nil {
		t.Fatalf("未配置密钥时应放行本地联调: %v", err)
	}
}

func TestVerifySignature_GitLabAndGiteeUseTokenComparison(t *testing.T) {
	service := Service{Webhook: config.WebhookConfig{GitLabSecret: "gitlab-token", GiteeSecret: "gitee-token"}}

	if err := service.VerifySignature(providerGitLab, "gitlab-token", nil); err != nil {
		t.Fatalf("GitLab 令牌正确时应通过: %v", err)
	}
	if err := service.VerifySignature(providerGitLab, "wrong", nil); err == nil {
		t.Fatalf("GitLab 令牌错误时应拒绝")
	}
	if err := service.VerifySignature(providerGitee, "gitee-token", nil); err != nil {
		t.Fatalf("Gitee 令牌正确时应通过: %v", err)
	}
	if err := service.VerifySignature(providerGitee, "wrong", nil); err == nil {
		t.Fatalf("Gitee 令牌错误时应拒绝")
	}
}

func TestDetectEventType_MapsProviders(t *testing.T) {
	for _, test := range []struct {
		header       string
		wantProvider string
		wantEvent    string
	}{
		{header: "X-Gitlab-Push Hook", wantProvider: providerGitLab, wantEvent: "Push Hook"},
		{header: "X-Gitee-Push Hook", wantProvider: providerGitee, wantEvent: "Push Hook"},
		{header: "Push Hook", wantProvider: providerGitHub, wantEvent: "Push Hook"},
		{header: "", wantProvider: providerGitHub, wantEvent: "push"},
	} {
		t.Run(test.header, func(t *testing.T) {
			provider, event := detectEventType(test.header)
			if provider != test.wantProvider || event != test.wantEvent {
				t.Fatalf("解析结果应为 (%s, %s)，实际为 (%s, %s)", test.wantProvider, test.wantEvent, provider, event)
			}
		})
	}
}

func TestParseOptionalUUID_EmptyAndInvalid(t *testing.T) {
	if id, err := parseOptionalUUID(""); err != nil || id != uuid.Nil {
		t.Fatalf("空字符串应返回空 UUID，实际为 (%s, %v)", id, err)
	}
	if _, err := parseOptionalUUID("not-a-uuid"); err == nil {
		t.Fatalf("非法 UUID 应返回错误")
	}
	valid := uuid.New().String()
	if id, err := parseOptionalUUID(valid); err != nil || id.String() != valid {
		t.Fatalf("合法 UUID 解析失败: (%s, %v)", id, err)
	}
}

func TestToSuggestionView_ParsesContent(t *testing.T) {
	content := []byte(`{"summary":"已完成登录","requirements":[{"code":"REQ-001","title":"登录"}],"impactNotes":["影响验收"]}`)
	model := db.AiSuggestion{
		Content:   content,
		CreatedAt: time.Date(2026, 11, 26, 10, 0, 0, 0, time.UTC),
		ID:        uuid.New(),
		Kind:      "requirements",
		Model:     pgtype.Text{String: "test-model", Valid: true},
		Status:    "draft",
	}

	view := toSuggestionView(model)

	if view.Content.Summary != "已完成登录" {
		t.Fatalf("摘要解析失败: %q", view.Content.Summary)
	}
	if len(view.Content.Requirements) != 1 || view.Content.Requirements[0].Code != "REQ-001" {
		t.Fatalf("候选需求解析失败: %+v", view.Content.Requirements)
	}
	if len(view.Content.ImpactNotes) != 1 || view.Content.ImpactNotes[0] != "影响验收" {
		t.Fatalf("影响说明解析失败: %+v", view.Content.ImpactNotes)
	}
}

func TestHmacSHA256_IsDeterministic(t *testing.T) {
	first := hmacSHA256("secret", []byte("payload"))
	second := hmacSHA256("secret", []byte("payload"))
	if !hmac.Equal(first, second) {
		t.Fatalf("同一密钥与载荷的 HMAC 应稳定")
	}
	expected := sha256.Sum256([]byte("payload"))
	if hmac.Equal(first, expected[:]) {
		t.Fatalf("HMAC 不应等于裸 SHA-256")
	}
}

func TestService_ZeroValueDoesNotPanic(t *testing.T) {
	service := Service{}
	if err := service.VerifySignature("unknown", "sig", []byte("payload")); err != nil {
		t.Fatalf("未知来源且无密钥配置时应放行: %v", err)
	}
}

func TestReceiveWebhook_DoesNotCallDatabaseOnBadSignature(t *testing.T) {
	service := Service{Webhook: config.WebhookConfig{GitHubSecret: "secret"}}
	_, err := service.ReceiveWebhook(context.Background(), providerGitHub, "Push Hook", "bad", []byte(`{}`), nil)
	if err == nil {
		t.Fatalf("签名错误时应拒绝入库")
	}
}
