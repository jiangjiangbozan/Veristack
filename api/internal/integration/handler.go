// Package integration 的 Handler 层，负责外部接入接口的数据绑定、权限校验与响应构造。
package integration

import (
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"api/internal/platform/access"
	"api/internal/platform/audit"
	"api/internal/platform/db"
	"api/internal/platform/dbutil"
	"api/internal/platform/errs"
	"api/internal/platform/idparse"
	"api/internal/platform/response"
	"api/internal/platform/validate"
)

const (
	// eventTypeHeaderGitHub 是 GitHub 推送事件类型的事件头。
	eventTypeHeaderGitHub = "X-GitHub-Event"
	// eventTypeHeaderGitLab 是 GitLab 推送事件类型的事件头。
	eventTypeHeaderGitLab = "X-Gitlab-Event"
	// signatureHeaderGitHub 是 GitHub 事件签名头。
	signatureHeaderGitHub = "X-Hub-Signature-256"
	// tokenHeaderGitLab 是 GitLab 事件令牌头。
	tokenHeaderGitLab = "X-Gitlab-Token"
	// eventTypeHeaderGitee 是 Gitee 事件类型头。
	eventTypeHeaderGitee = "X-Gitee-Event"
	// tokenHeaderGitee 是 Gitee 事件令牌头。
	tokenHeaderGitee = "X-Gitee-Token"
	// maxWebhookBodyBytes 限制 Webhook 请求体大小，防止超大载荷占用内存。
	maxWebhookBodyBytes = 2 << 20
)

// Handler 提供外部集成相关的 HTTP 接口。
type Handler struct {
	Service *Service // 集成业务服务
}

// PublicRoutes 注册无需登录的第三方回调接口，由签名或令牌保证可信。
// 项目 ID 放在 URL 中，便于第三方为每个项目配置独立的回调地址。
func (h *Handler) PublicRoutes(r chi.Router) {
	r.Post("/webhooks/{provider}/projects/{projectId}", h.receive)
}

// Routes 注册开发者使用的集成接口，全部需要鉴权。
func (h *Handler) Routes(r chi.Router) {
	r.Post("/projects/{projectId}/ci-reports", h.importCIReport)
	r.Get("/projects/{projectId}/ci-reports", h.listCIReports)
	r.Get("/projects/{projectId}/webhooks", h.listWebhooks)
	r.Post("/projects/{projectId}/ai/requirements", h.extractRequirements)
	r.Post("/projects/{projectId}/ai/change-impact", h.analyzeChangeImpact)
	r.Post("/projects/{projectId}/ai/delivery-summary", h.generateSummary)
	r.Get("/projects/{projectId}/ai/suggestions", h.listSuggestions)
	r.Post("/ai/suggestions/{suggestionId}/apply", h.applySuggestion)
	r.Post("/ai/suggestions/{suggestionId}/dismiss", h.dismissSuggestion)
}

// analyzeChangeImpact 生成变更影响分析建议，结果仅为草稿。
func (h *Handler) analyzeChangeImpact(w http.ResponseWriter, r *http.Request) {
	projectID, principal, ok := h.resolveProject(w, r)
	if !ok {
		return
	}
	var req AnalyzeChangeImpactRequest
	if err := validate.Decode(r, &req); err != nil {
		response.Fail(w, errs.BadRequest(err.Error(), err))
		return
	}
	change, err := db.New(h.Service.Database).GetChangeRequest(r.Context(), uuid.MustParse(req.ChangeRequestID))
	if err != nil {
		response.Fail(w, errs.NotFound("变更申请不存在", err))
		return
	}
	baseline, err := db.New(h.Service.Database).GetScopeBaseline(r.Context(), change.BaselineID)
	if err != nil {
		response.Fail(w, errs.NotFound("需求基线不存在", err))
		return
	}
	suggestion, err := h.Service.AnalyzeChangeImpact(r.Context(), projectID, string(baseline.RequirementsJson), change.Reason)
	if err != nil {
		response.Fail(w, err)
		return
	}
	h.recordAudit(r, principal, "integration.ai.change_impact", suggestion.ID, nil, suggestion)
	response.OK(w, AnalyzeChangeImpactResponse{Suggestion: toSuggestionView(suggestion)})
}

// applySuggestion 采纳一条 AI 建议。
func (h *Handler) applySuggestion(w http.ResponseWriter, r *http.Request) {
	suggestionID, principal, ok := h.resolveSuggestion(w, r)
	if !ok {
		return
	}
	suggestion, err := h.Service.ApplySuggestion(r.Context(), suggestionID)
	if err != nil {
		response.Fail(w, err)
		return
	}
	h.recordAudit(r, principal, "integration.ai.apply", suggestion.ID, nil, suggestion)
	response.OK(w, toSuggestionView(suggestion))
}

// dismissSuggestion 忽略一条 AI 建议。
func (h *Handler) dismissSuggestion(w http.ResponseWriter, r *http.Request) {
	suggestionID, principal, ok := h.resolveSuggestion(w, r)
	if !ok {
		return
	}
	suggestion, err := h.Service.DismissSuggestion(r.Context(), suggestionID)
	if err != nil {
		response.Fail(w, err)
		return
	}
	h.recordAudit(r, principal, "integration.ai.dismiss", suggestion.ID, nil, suggestion)
	response.OK(w, toSuggestionView(suggestion))
}

// extractRequirements 从需求文档提取候选需求建议，结果仅为草稿。
func (h *Handler) extractRequirements(w http.ResponseWriter, r *http.Request) {
	projectID, principal, ok := h.resolveProject(w, r)
	if !ok {
		return
	}
	var req ExtractRequirementsRequest
	if err := validate.Decode(r, &req); err != nil {
		response.Fail(w, errs.BadRequest(err.Error(), err))
		return
	}
	suggestion, err := h.Service.ExtractRequirements(r.Context(), projectID, req.Content)
	if err != nil {
		response.Fail(w, err)
		return
	}
	h.recordAudit(r, principal, "integration.ai.extract", suggestion.ID, nil, suggestion)
	response.OK(w, ExtractRequirementsResponse{Suggestion: toSuggestionView(suggestion)})
}

// generateSummary 为交付版本生成自然语言摘要建议。
func (h *Handler) generateSummary(w http.ResponseWriter, r *http.Request) {
	projectID, principal, ok := h.resolveProject(w, r)
	if !ok {
		return
	}
	var req GenerateSummaryRequest
	if err := validate.Decode(r, &req); err != nil {
		response.Fail(w, errs.BadRequest(err.Error(), err))
		return
	}
	deliveryID, err := uuid.Parse(req.DeliveryID)
	if err != nil {
		response.Fail(w, errs.BadRequest("交付版本 ID 无效", err))
		return
	}
	delivery, err := db.New(h.Service.Database).GetDelivery(r.Context(), deliveryID)
	if err != nil {
		response.Fail(w, errs.NotFound("交付版本不存在", err))
		return
	}
	suggestion, err := h.Service.GenerateDeliverySummary(r.Context(), projectID, string(delivery.ManifestJson))
	if err != nil {
		response.Fail(w, err)
		return
	}
	h.recordAudit(r, principal, "integration.ai.summary", suggestion.ID, nil, suggestion)
	response.OK(w, GenerateSummaryResponse{Suggestion: toSuggestionView(suggestion)})
}

// importCIReport 导入 CI 测试报告并汇总结果。
func (h *Handler) importCIReport(w http.ResponseWriter, r *http.Request) {
	projectID, principal, ok := h.resolveProject(w, r)
	if !ok {
		return
	}
	var req ImportCIReportRequest
	if err := validate.Decode(r, &req); err != nil {
		response.Fail(w, errs.BadRequest(err.Error(), err))
		return
	}
	report, summary, err := h.Service.ImportCIReport(r.Context(), projectID, req)
	if err != nil {
		response.Fail(w, err)
		return
	}
	h.recordAudit(r, principal, "integration.ci.import", report.ID, nil, report)
	response.OK(w, ImportCIReportResponse{
		Report:  CIReportView{BuildNo: req.BuildNo, CreatedAt: dbutil.TimeString(report.CreatedAt), ID: report.ID.String(), Provider: report.Provider},
		Summary: TestSummaryView{DurationMs: summary.TotalDuration().Milliseconds(), Failed: summary.Failed, Passed: summary.Passed, Skipped: summary.Skipped, Total: summary.Total},
	})
}

// listCIReports 列出项目导入的 CI 报告。
func (h *Handler) listCIReports(w http.ResponseWriter, r *http.Request) {
	projectID, _, ok := h.resolveProject(w, r)
	if !ok {
		return
	}
	pagination := response.ParsePagination(r)
	reports, err := db.New(h.Service.Database).ListCIReportsByProject(r.Context(), db.ListCIReportsByProjectParams{
		Limit:     int32(pagination.PageSize),
		Offset:    int32((pagination.Page - 1) * pagination.PageSize),
		ProjectID: projectID,
	})
	if err != nil {
		response.Fail(w, err)
		return
	}
	total, err := db.New(h.Service.Database).CountCIReportsByProject(r.Context(), projectID)
	if err != nil {
		response.Fail(w, err)
		return
	}
	items := make([]CIReportView, 0, len(reports))
	for _, report := range reports {
		items = append(items, CIReportView{
			BuildNo:   report.BuildNo.String,
			CreatedAt: dbutil.TimeString(report.CreatedAt),
			ID:        report.ID.String(),
			Provider:  report.Provider,
		})
	}
	response.OKWithPage(w, ListCIReportsResponse{Items: items}, response.PageOf(pagination, total))
}

// listSuggestions 列出项目的 AI 建议草稿。
func (h *Handler) listSuggestions(w http.ResponseWriter, r *http.Request) {
	projectID, _, ok := h.resolveProject(w, r)
	if !ok {
		return
	}
	kind := r.URL.Query().Get("kind")
	pagination := response.ParsePagination(r)
	suggestions, err := h.Service.ListSuggestions(r.Context(), projectID, kind, int32(pagination.PageSize), int32((pagination.Page-1)*pagination.PageSize))
	if err != nil {
		response.Fail(w, err)
		return
	}
	items := make([]AISuggestionView, 0, len(suggestions))
	for _, suggestion := range suggestions {
		items = append(items, toSuggestionView(suggestion))
	}
	response.OK(w, ListSuggestionsResponse{Items: items})
}

// listWebhooks 列出项目收到的 Webhook 事件。
func (h *Handler) listWebhooks(w http.ResponseWriter, r *http.Request) {
	projectID, _, ok := h.resolveProject(w, r)
	if !ok {
		return
	}
	pagination := response.ParsePagination(r)
	events, err := h.Service.ListWebhooks(r.Context(), projectID, int32(pagination.PageSize), int32((pagination.Page-1)*pagination.PageSize))
	if err != nil {
		response.Fail(w, err)
		return
	}
	items := make([]WebhookView, 0, len(events))
	for _, event := range events {
		items = append(items, WebhookView{
			EventType:  event.EventType,
			ID:         event.ID.String(),
			Provider:   event.Provider,
			ReceivedAt: dbutil.TimeString(event.ReceivedAt),
			Status:     event.Status,
		})
	}
	response.OK(w, ListWebhooksResponse{Items: items})
}

// receive 接收第三方 Webhook 事件，先落库再异步处理。
func (h *Handler) receive(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")
	projectID, err := idparse.FromURL(r, "projectId")
	if err != nil {
		response.Fail(w, errs.BadRequest(err.Error(), err))
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, maxWebhookBodyBytes))
	if err != nil {
		response.Fail(w, errs.BadRequest("读取事件内容失败", err))
		return
	}
	headerEvent := r.Header.Get(eventTypeHeaderGitHub)
	if headerEvent == "" {
		headerEvent = r.Header.Get(eventTypeHeaderGitLab)
	}
	if headerEvent == "" {
		headerEvent = r.Header.Get(eventTypeHeaderGitee)
	}
	detectedProvider, eventType := detectEventType(headerEvent)
	if provider != "" && provider != detectedProvider {
		detectedProvider = provider
	}
	signature := r.Header.Get(signatureHeaderGitHub)
	if signature == "" {
		signature = r.Header.Get(tokenHeaderGitLab)
	}
	if signature == "" {
		signature = r.Header.Get(tokenHeaderGitee)
	}
	event, err := h.Service.ReceiveWebhook(r.Context(), detectedProvider, eventType, signature, body, &projectID)
	if err != nil {
		response.Fail(w, err)
		return
	}
	response.OK(w, ReceiveWebhookResponse{
		Accepted: true,
		Webhook: WebhookView{
			EventType:  event.EventType,
			ID:         event.ID.String(),
			Provider:   event.Provider,
			ReceivedAt: dbutil.TimeString(event.ReceivedAt),
			Status:     event.Status,
		},
	})
}

// recordAudit 记录集成写操作的审计事件。
func (h *Handler) recordAudit(r *http.Request, principal access.Principal, action string, objectID uuid.UUID, before, after any) {
	writer := audit.Writer{Queries: db.New(h.Service.Database)}
	_ = writer.Record(r.Context(), audit.FromRequest(r, principal.ID), action, "integration", objectID, before, after)
}

// resolveProject 解析路径中的项目 ID 并校验组织归属。
func (h *Handler) resolveProject(w http.ResponseWriter, r *http.Request) (uuid.UUID, access.Principal, bool) {
	projectID, err := idparse.FromURL(r, "projectId")
	if err != nil {
		response.Fail(w, errs.BadRequest(err.Error(), err))
		return uuid.Nil, access.Principal{}, false
	}
	queries := db.New(h.Service.Database)
	principal, err := access.FromRequest(r.Context(), r, queries)
	if err != nil {
		response.Fail(w, err)
		return uuid.Nil, access.Principal{}, false
	}
	project, err := queries.GetProject(r.Context(), projectID)
	if err != nil {
		response.Fail(w, errs.NotFound("项目不存在", err))
		return uuid.Nil, access.Principal{}, false
	}
	if err := access.RequireSameOrganization(principal, project.OrganizationID); err != nil {
		response.Fail(w, err)
		return uuid.Nil, access.Principal{}, false
	}
	return projectID, principal, true
}

// resolveSuggestion 解析路径中的建议 ID 并校验资源归属。
func (h *Handler) resolveSuggestion(w http.ResponseWriter, r *http.Request) (uuid.UUID, access.Principal, bool) {
	suggestionID, err := idparse.FromURL(r, "suggestionId")
	if err != nil {
		response.Fail(w, errs.BadRequest(err.Error(), err))
		return uuid.Nil, access.Principal{}, false
	}
	queries := db.New(h.Service.Database)
	principal, err := access.FromRequest(r.Context(), r, queries)
	if err != nil {
		response.Fail(w, err)
		return uuid.Nil, access.Principal{}, false
	}
	suggestion, err := queries.GetAISuggestion(r.Context(), suggestionID)
	if err != nil {
		response.Fail(w, errs.NotFound("AI 建议不存在", err))
		return uuid.Nil, access.Principal{}, false
	}
	project, err := queries.GetProject(r.Context(), suggestion.ProjectID)
	if err != nil {
		response.Fail(w, errs.NotFound("项目不存在", err))
		return uuid.Nil, access.Principal{}, false
	}
	if err := access.RequireSameOrganization(principal, project.OrganizationID); err != nil {
		response.Fail(w, err)
		return uuid.Nil, access.Principal{}, false
	}
	return suggestionID, principal, true
}
