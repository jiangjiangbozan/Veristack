// Package acceptance 的 Handler 层，负责验收与异议接口的数据绑定、权限校验与响应构造。
package acceptance

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"api/internal/evidence"
	"api/internal/identity"
	"api/internal/platform/access"
	"api/internal/platform/audit"
	"api/internal/platform/db"
	"api/internal/platform/errs"
	"api/internal/platform/idparse"
	"api/internal/platform/response"
	"api/internal/platform/validate"
	"api/internal/platform/views"
)

const (
	// portalPath 是客户验收门户的路径前缀，该前缀下的接口不要求登录。
	portalPath = "/share/{token}"
	// portalPathPrefix 是前端客户门户的访问路径前缀。
	portalPathPrefix = "/accept/"
	// timeLayout 是对外暴露时间的统一格式。
	timeLayout = time.RFC3339
)

// Handler 提供验收相关的 HTTP 接口。
type Handler struct {
	Cooldown        *identity.CooldownLimiter // 验证码发送冷却限流器
	EvidenceService *evidence.Service         // 证据业务服务，用于客户门户展示证据
	Identity        *CustomerIdentityService  // 客户邮箱验证服务
	Service         *Service                  // 验收业务服务
}

// PortalRoutes 注册客户验收门户接口，通过分享链接令牌访问。
func (h *Handler) PortalRoutes(r chi.Router) {
	r.Route(portalPath, func(r chi.Router) {
		r.Get("/", h.portal)
		r.Post("/verification-code", h.requestVerificationCode)
		r.Post("/verify", h.verifyCustomer)
		r.Post("/verdicts/{criterionKey}", h.portalVerdict)
		r.Post("/decision", h.portalDecision)
	})
}

// Routes 注册开发者视角的验收接口，全部需要鉴权。
func (h *Handler) Routes(r chi.Router) {
	r.Post("/deliveries/{deliveryId}/share-token", h.createShareToken)
	r.Get("/deliveries/{deliveryId}/acceptance", h.summary)
	r.Post("/objections/{objectionId}/replies", h.reply)
	r.Post("/objections/{objectionId}/resolve", h.resolve)
}

// requestVerificationCode 为客户签发邮箱验证码。
//
// 验证码只发送一次且有时效，数据库仅保存摘要。
func (h *Handler) requestVerificationCode(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	if _, err := h.Service.GetDeliveryByToken(r.Context(), token); err != nil {
		response.Fail(w, err)
		return
	}
	var req RequestVerificationCodeRequest
	if err := validate.Decode(r, &req); err != nil {
		response.Fail(w, errs.BadRequest(err.Error(), err))
		return
	}
	if h.Cooldown != nil && !h.Cooldown.Allow("acceptance-code|"+strings.ToLower(strings.TrimSpace(req.Email))) {
		response.Fail(w, errs.TooManyRequests("验证码发送过于频繁，请稍后再试", nil))
		return
	}
	if err := h.Identity.RequestVerificationCode(r.Context(), req.Email); err != nil {
		response.Fail(w, err)
		return
	}
	response.OK(w, RequestVerificationCodeResponse{Sent: true})
}

// verifyCustomer 校验客户提交的邮箱验证码并登记已验证身份。
func (h *Handler) verifyCustomer(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	delivery, err := h.Service.GetDeliveryByToken(r.Context(), token)
	if err != nil {
		response.Fail(w, err)
		return
	}
	var req VerifyCustomerRequest
	if err := validate.Decode(r, &req); err != nil {
		response.Fail(w, errs.BadRequest(err.Error(), err))
		return
	}
	if err := h.Service.RegisterVerifiedIdentity(r.Context(), delivery.ID, req.Email, req.Code); err != nil {
		response.Fail(w, err)
		return
	}
	response.OK(w, VerifyCustomerResponse{Verified: true})
}

// portalDecision 记录客户在门户提交的最终验收结论。
//
// 客户必须先完成邮箱验证，否则结论无法归因到可追溯的身份。
func (h *Handler) portalDecision(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	delivery, err := h.Service.GetDeliveryByToken(r.Context(), token)
	if err != nil {
		response.Fail(w, err)
		return
	}
	var req SubmitDecisionRequest
	if err := validate.Decode(r, &req); err != nil {
		response.Fail(w, errs.BadRequest(err.Error(), err))
		return
	}
	if _, err := h.Service.EnsureVerifiedCustomer(r.Context(), delivery.ID, req.Email); err != nil {
		response.Fail(w, err)
		return
	}
	updated, err := h.Service.SubmitDecision(r.Context(), delivery.ID, req)
	if err != nil {
		response.Fail(w, err)
		return
	}
	response.OK(w, SubmitDecisionResponse{Delivery: views.FromDelivery(updated)})
}

// portalVerdict 保存客户在门户对单条验收条件的裁定。
//
// 客户必须先完成邮箱验证，否则裁定无法归因到可追溯的身份。
func (h *Handler) portalVerdict(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	delivery, err := h.Service.GetDeliveryByToken(r.Context(), token)
	if err != nil {
		response.Fail(w, err)
		return
	}
	var req SaveVerdictRequest
	if err := validate.Decode(r, &req); err != nil {
		response.Fail(w, errs.BadRequest(err.Error(), err))
		return
	}
	customer, err := h.Service.EnsureVerifiedCustomer(r.Context(), delivery.ID, req.Email)
	if err != nil {
		response.Fail(w, err)
		return
	}
	verdict, err := h.Service.SaveVerdict(r.Context(), delivery.ID, customer.ID, chi.URLParam(r, "criterionKey"), req.Verdict, req.Comment)
	if err != nil {
		response.Fail(w, err)
		return
	}
	response.OK(w, SaveVerdictResponse{Verdict: views.FromAcceptanceVerdict(verdict)})
}

// portal 返回客户可见的交付内容、验收条件与已有裁定。
func (h *Handler) portal(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	delivery, err := h.Service.GetDeliveryByToken(r.Context(), token)
	if err != nil {
		response.Fail(w, err)
		return
	}
	verdicts, err := h.Service.ListVerdicts(r.Context(), delivery.ID)
	if err != nil {
		response.Fail(w, err)
		return
	}
	evidenceItems, err := h.EvidenceService.ListByDelivery(r.Context(), delivery.ID)
	if err != nil {
		response.Fail(w, err)
		return
	}
	evidenceViews := make([]views.EvidenceView, 0, len(evidenceItems))
	for _, item := range evidenceItems {
		evidenceViews = append(evidenceViews, views.FromEvidence(item))
	}
	response.OK(w, PortalDeliveryResponse{
		Delivery: views.FromDelivery(delivery),
		Evidence: evidenceViews,
		Manifest: views.FromManifest(delivery.ManifestJson),
		Verdicts: verdicts,
	})
}

// createShareToken 生成客户验收分享链接。
//
// 令牌签发与审计事件在同一事务内完成，审计失败会连带撤销令牌签发。
func (h *Handler) createShareToken(w http.ResponseWriter, r *http.Request) {
	deliveryID, err := idparse.FromURL(r, "deliveryId")
	if err != nil {
		response.Fail(w, errs.BadRequest(err.Error(), err))
		return
	}
	principal, queries, err := h.principal(r)
	if err != nil {
		response.Fail(w, err)
		return
	}
	if _, err := h.ensureDelivery(w, r, queries, deliveryID, principal); err != nil {
		return
	}
	var req CreateShareTokenRequest
	if err := validate.Decode(r, &req); err != nil {
		response.Fail(w, errs.BadRequest(err.Error(), err))
		return
	}
	token, expiresAt, err := h.Service.CreateShareToken(r.Context(), deliveryID, req.ExpiresInDays)
	if err != nil {
		response.Fail(w, err)
		return
	}
	// 审计只记录令牌已签发的事实与有效期，绝不记录令牌本身，
	// 否则数据库泄露等同于所有客户验收链接泄露。
	if err := h.recordAudit(r.Context(), r, principal, "acceptance.share", deliveryID, nil, map[string]any{
		"expiresAt":   expiresAt.Format(timeLayout),
		"tokenIssued": true,
	}); err != nil {
		response.Fail(w, err)
		return
	}
	response.OK(w, CreateShareTokenResponse{
		ExpiresAt: expiresAt.Format(timeLayout),
		Token:     token,
		URL:       portalPathPrefix + token,
	})
}

// ensureDelivery 校验交付归属，失败时写入错误响应。
func (h *Handler) ensureDelivery(w http.ResponseWriter, r *http.Request, queries *db.Queries, deliveryID uuid.UUID, principal access.Principal) (db.Delivery, error) {
	delivery, err := queries.GetDelivery(r.Context(), deliveryID)
	if err != nil {
		response.Fail(w, errs.NotFound("交付版本不存在", err))
		return db.Delivery{}, err
	}
	project, err := queries.GetProject(r.Context(), delivery.ProjectID)
	if err != nil {
		response.Fail(w, errs.NotFound("项目不存在", err))
		return db.Delivery{}, err
	}
	if err := access.RequireSameOrganization(principal, project.OrganizationID); err != nil {
		response.Fail(w, err)
		return db.Delivery{}, err
	}
	return delivery, nil
}

// principal 解析当前访问主体与查询对象。
func (h *Handler) principal(r *http.Request) (access.Principal, *db.Queries, error) {
	queries := db.New(h.Service.Database)
	principal, err := access.FromRequest(r.Context(), r, queries)
	return principal, queries, err
}

// reply 开发者回复客户异议，形成闭环。
func (h *Handler) reply(w http.ResponseWriter, r *http.Request) {
	objectionID, err := idparse.FromURL(r, "objectionId")
	if err != nil {
		response.Fail(w, errs.BadRequest(err.Error(), err))
		return
	}
	principal, queries, err := h.principal(r)
	if err != nil {
		response.Fail(w, err)
		return
	}
	if err := h.ensureObjectionDelivery(w, r, queries, objectionID, principal); err != nil {
		return
	}
	var req ReplyObjectionRequest
	if err := validate.Decode(r, &req); err != nil {
		response.Fail(w, errs.BadRequest(err.Error(), err))
		return
	}
	reply, err := h.Service.ReplyToObjection(r.Context(), objectionID, principal.ID, req.Content)
	if err != nil {
		response.Fail(w, err)
		return
	}
	if err := h.recordAudit(r.Context(), r, principal, "acceptance.reply", objectionID, nil, reply); err != nil {
		response.Fail(w, err)
		return
	}
	response.OK(w, ReplyObjectionResponse{Reply: views.FromObjectionReply(reply)})
}

// ensureObjectionDelivery 校验异议所属交付的组织归属。
func (h *Handler) ensureObjectionDelivery(w http.ResponseWriter, r *http.Request, queries *db.Queries, objectionID uuid.UUID, principal access.Principal) error {
	objection, err := queries.GetObjection(r.Context(), objectionID)
	if err != nil {
		response.Fail(w, errs.NotFound("异议不存在", err))
		return err
	}
	if _, err := h.ensureDelivery(w, r, queries, objection.DeliveryID, principal); err != nil {
		return err
	}
	return nil
}

// resolve 关闭客户异议，标记客户已复核。
func (h *Handler) resolve(w http.ResponseWriter, r *http.Request) {
	objectionID, err := idparse.FromURL(r, "objectionId")
	if err != nil {
		response.Fail(w, errs.BadRequest(err.Error(), err))
		return
	}
	principal, queries, err := h.principal(r)
	if err != nil {
		response.Fail(w, err)
		return
	}
	if err := h.ensureObjectionDelivery(w, r, queries, objectionID, principal); err != nil {
		return
	}
	objection, err := h.Service.ResolveObjection(r.Context(), objectionID)
	if err != nil {
		response.Fail(w, err)
		return
	}
	if err := h.recordAudit(r.Context(), r, principal, "acceptance.resolve", objectionID, nil, objection); err != nil {
		response.Fail(w, err)
		return
	}
	response.OK(w, ResolveObjectionResponse{Objection: views.FromObjection(objection, nil)})
}

// summary 返回开发者视角的验收概览：裁定、异议与交付状态。
func (h *Handler) summary(w http.ResponseWriter, r *http.Request) {
	deliveryID, err := idparse.FromURL(r, "deliveryId")
	if err != nil {
		response.Fail(w, errs.BadRequest(err.Error(), err))
		return
	}
	principal, queries, err := h.principal(r)
	if err != nil {
		response.Fail(w, err)
		return
	}
	delivery, err := h.ensureDelivery(w, r, queries, deliveryID, principal)
	if err != nil {
		return
	}
	verdicts, err := h.Service.ListVerdicts(r.Context(), delivery.ID)
	if err != nil {
		response.Fail(w, err)
		return
	}
	objections, err := h.Service.ListObjections(r.Context(), delivery.ID)
	if err != nil {
		response.Fail(w, err)
		return
	}
	response.OK(w, AcceptanceSummaryResponse{
		Delivery:   views.FromDelivery(delivery),
		Objections: objections,
		Verdicts:   verdicts,
	})
}

// recordAudit 在当前请求事务内写入审计事件，写入失败会中断请求并回滚业务变更。
func (h *Handler) recordAudit(ctx context.Context, r *http.Request, principal access.Principal, action string, objectID uuid.UUID, before, after any) error {
	writer := audit.WriterFor(ctx, h.Service.Database)
	return writer.Record(ctx, audit.FromRequest(r, principal.ID), action, "acceptance", objectID, before, after)
}
