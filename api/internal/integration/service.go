// Package integration 的业务服务，负责 Webhook 接收与处理、CI 报告导入与 AI 建议生成。
package integration

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"api/internal/platform/ai"
	"api/internal/platform/config"
	"api/internal/platform/db"
	"api/internal/platform/dbutil"
	"api/internal/platform/errs"
	"api/internal/platform/testreport"
)

const (
	providerGitHub = "github"
	providerGitLab = "gitlab"
	providerGitee  = "gitee"
)

// Service 提供外部集成相关的业务方法。
type Service struct {
	AI       ai.Model             // 模型适配器
	Database *pgxpool.Pool        // 数据库连接池
	Webhook  config.WebhookConfig // Webhook 校验密钥
}

// ApplySuggestion 采纳一条 AI 建议，仅改变建议状态，不直接改写正式基线。
func (s *Service) ApplySuggestion(ctx context.Context, id uuid.UUID) (db.AiSuggestion, error) {
	suggestion, err := db.New(s.Database).MarkAISuggestionApplied(ctx, id)
	if err != nil {
		return db.AiSuggestion{}, errs.NotFound("AI 建议不存在", err)
	}
	return suggestion, nil
}

// DismissSuggestion 忽略一条 AI 建议，保留记录以便追溯。
func (s *Service) DismissSuggestion(ctx context.Context, id uuid.UUID) (db.AiSuggestion, error) {
	suggestion, err := db.New(s.Database).MarkAISuggestionDismissed(ctx, id)
	if err != nil {
		return db.AiSuggestion{}, errs.NotFound("AI 建议不存在", err)
	}
	return suggestion, nil
}

// ExtractRequirements 从需求文档生成候选需求建议，结果仅为草稿。
func (s *Service) ExtractRequirements(ctx context.Context, projectID uuid.UUID, content string) (db.AiSuggestion, error) {
	result, err := s.AI.ExtractRequirements(ctx, content)
	if err != nil {
		return db.AiSuggestion{}, fmt.Errorf("生成需求提取建议失败: %w", err)
	}
	return s.saveSuggestion(ctx, projectID, string(ai.SuggestionKindRequirements), result)
}

// GenerateDeliverySummary 为交付版本生成自然语言摘要建议，结果仅为草稿。
func (s *Service) GenerateDeliverySummary(ctx context.Context, projectID uuid.UUID, deliveryText string) (db.AiSuggestion, error) {
	result, err := s.AI.SummarizeDelivery(ctx, deliveryText)
	if err != nil {
		return db.AiSuggestion{}, fmt.Errorf("生成交付摘要建议失败: %w", err)
	}
	return s.saveSuggestion(ctx, projectID, string(ai.SuggestionKindDeliverySummary), result)
}

// AnalyzeChangeImpact 分析变更对既有需求的影响，产出影响说明建议。
func (s *Service) AnalyzeChangeImpact(ctx context.Context, projectID uuid.UUID, baselineText, changeText string) (db.AiSuggestion, error) {
	result, err := s.AI.AnalyzeChangeImpact(ctx, baselineText, changeText)
	if err != nil {
		return db.AiSuggestion{}, fmt.Errorf("生成变更影响建议失败: %w", err)
	}
	return s.saveSuggestion(ctx, projectID, string(ai.SuggestionKindChangeImpact), result)
}

// ImportCIReport 解析并保存 CI 测试报告。
func (s *Service) ImportCIReport(ctx context.Context, projectID uuid.UUID, req ImportCIReportRequest) (db.CiReport, testreport.Summary, error) {
	summary, err := testreport.ParseJUnit([]byte(req.ReportXML))
	if err != nil {
		return db.CiReport{}, testreport.Summary{}, errs.BadRequest(err.Error(), err)
	}
	detail, err := dbutil.JSONBytes(summary.Cases)
	if err != nil {
		return db.CiReport{}, testreport.Summary{}, err
	}
	deliveryID, err := parseOptionalUUID(req.DeliveryID)
	if err != nil {
		return db.CiReport{}, testreport.Summary{}, errs.BadRequest("交付版本 ID 无效", err)
	}
	report, err := db.New(s.Database).CreateCIReport(ctx, db.CreateCIReportParams{
		BuildNo:      dbutil.TextFromString(req.BuildNo),
		DeliveryID:   dbutil.NullableUUID(deliveryID),
		DurationMs:   dbutil.NullableInt64(summary.TotalDuration().Milliseconds()),
		FailedCount:  int32(summary.Failed),
		GitRef:       dbutil.TextFromString(req.GitRef),
		PassedCount:  int32(summary.Passed),
		ProjectID:    projectID,
		Provider:     req.Provider,
		SkippedCount: int32(summary.Skipped),
		Summary:      detail,
		TotalCount:   int32(summary.Total),
	})
	if err != nil {
		return db.CiReport{}, testreport.Summary{}, fmt.Errorf("保存 CI 报告失败: %w", err)
	}
	return report, summary, nil
}

// ListSuggestions 列出项目的 AI 建议草稿。
func (s *Service) ListSuggestions(ctx context.Context, projectID uuid.UUID, kind string, limit, offset int32) ([]db.AiSuggestion, error) {
	return db.New(s.Database).ListAISuggestionsByProject(ctx, db.ListAISuggestionsByProjectParams{
		Kind:      kind,
		Limit:     limit,
		Offset:    offset,
		ProjectID: projectID,
	})
}

// ListWebhooks 列出项目收到的 Webhook 事件。
func (s *Service) ListWebhooks(ctx context.Context, projectID uuid.UUID, limit, offset int32) ([]db.Webhook, error) {
	return db.New(s.Database).ListWebhooksByProject(ctx, db.ListWebhooksByProjectParams{
		Limit:     limit,
		Offset:    offset,
		ProjectID: dbutil.NullableUUID(projectID),
	})
}

// ReceiveWebhook 校验签名后保存原始事件，并创建异步处理任务。
func (s *Service) ReceiveWebhook(ctx context.Context, provider, eventType, signature string, payload []byte, projectID *uuid.UUID) (db.Webhook, error) {
	if err := s.VerifySignature(provider, signature, payload); err != nil {
		return db.Webhook{}, err
	}
	queries := db.New(s.Database)
	event := db.CreateWebhookParams{
		EventType: eventType,
		Payload:   payload,
		Provider:  provider,
		Signature: dbutil.TextFromString(signature),
	}
	if projectID != nil {
		event.ProjectID = dbutil.NullableUUID(*projectID)
	}
	webhook, err := queries.CreateWebhook(ctx, event)
	if err != nil {
		return db.Webhook{}, fmt.Errorf("保存 Webhook 事件失败: %w", err)
	}
	task := db.CreateOutboxTaskParams{EventType: eventType, Payload: payload}
	if projectID != nil {
		task.ProjectID = dbutil.NullableUUID(*projectID)
	}
	if _, err := queries.CreateOutboxTask(ctx, task); err != nil {
		return db.Webhook{}, fmt.Errorf("创建 Webhook 处理任务失败: %w", err)
	}
	return webhook, nil
}

// VerifySignature 校验第三方 Webhook 的签名或令牌，密钥未配置时放行以便本地联调。
func (s *Service) VerifySignature(provider, signature string, payload []byte) error {
	switch provider {
	case providerGitHub:
		secret := s.Webhook.GitHubSecret
		if secret == "" {
			return nil
		}
		expected := "sha256=" + hex.EncodeToString(hmacSHA256(secret, payload))
		if !hmac.Equal([]byte(expected), []byte(signature)) {
			return errs.Unauthorized("GitHub 事件签名校验失败", nil)
		}
	case providerGitLab:
		secret := s.Webhook.GitLabSecret
		if secret == "" {
			return nil
		}
		if !hmac.Equal([]byte(secret), []byte(signature)) {
			return errs.Unauthorized("GitLab 事件令牌校验失败", nil)
		}
	case providerGitee:
		secret := s.Webhook.GiteeSecret
		if secret == "" {
			return nil
		}
		if !hmac.Equal([]byte(secret), []byte(signature)) {
			return errs.Unauthorized("Gitee 事件令牌校验失败", nil)
		}
	}
	return nil
}

// hmacSHA256 计算请求体的 HMAC-SHA256 摘要。
func hmacSHA256(secret string, payload []byte) []byte {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return mac.Sum(nil)
}

// parseOptionalUUID 解析可选 UUID，空字符串返回空值。
func parseOptionalUUID(raw string) (uuid.UUID, error) {
	if strings.TrimSpace(raw) == "" {
		return uuid.Nil, nil
	}
	return uuid.Parse(raw)
}

// detectEventType 从请求头推断事件类型与来源平台。
func detectEventType(headerEvent string) (string, string) {
	if strings.HasPrefix(headerEvent, "X-Gitlab") {
		return providerGitLab, strings.TrimPrefix(headerEvent, "X-Gitlab-")
	}
	if strings.HasPrefix(headerEvent, "X-Gitee") {
		return providerGitee, strings.TrimPrefix(headerEvent, "X-Gitee-")
	}
	if headerEvent != "" {
		return providerGitHub, headerEvent
	}
	return providerGitHub, "push"
}

// saveSuggestion 把模型产出保存为待确认的建议草稿。
func (s *Service) saveSuggestion(ctx context.Context, projectID uuid.UUID, kind string, result ai.Result) (db.AiSuggestion, error) {
	payload, err := json.Marshal(result)
	if err != nil {
		return db.AiSuggestion{}, fmt.Errorf("序列化模型建议失败: %w", err)
	}
	suggestion, err := db.New(s.Database).CreateAISuggestion(ctx, db.CreateAISuggestionParams{
		Content:   payload,
		Kind:      kind,
		Model:     dbutil.TextFromString(s.AI.Name()),
		ProjectID: projectID,
	})
	if err != nil {
		return db.AiSuggestion{}, fmt.Errorf("保存 AI 建议失败: %w", err)
	}
	return suggestion, nil
}

// toSuggestionView 把建议实体转换为对外视图。
func toSuggestionView(model db.AiSuggestion) AISuggestionView {
	content := SuggestionContent{ImpactNotes: []string{}, Requirements: []SuggestedRequirement{}}
	_ = dbutil.FromJSONBytes(model.Content, &content)
	return AISuggestionView{
		Content:   content,
		CreatedAt: dbutil.TimeString(model.CreatedAt),
		ID:        model.ID.String(),
		Kind:      model.Kind,
		Model:     dbutil.StringFromText(model.Model),
		Status:    model.Status,
	}
}
