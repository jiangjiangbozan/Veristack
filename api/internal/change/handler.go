// Package change 的 Handler 层，负责变更接口的数据绑定、权限校验与响应构造。
package change

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"api/internal/platform/access"
	"api/internal/platform/audit"
	"api/internal/platform/db"
	"api/internal/platform/errs"
	"api/internal/platform/idparse"
	"api/internal/platform/response"
	"api/internal/platform/validate"
	"api/internal/platform/views"
)

// Handler 提供变更申请相关的 HTTP 接口。
type Handler struct {
	Service *Service // 变更业务服务
}

// Routes 注册变更模块的路由，全部需要鉴权。
func (h *Handler) Routes(r chi.Router) {
	r.Route("/projects/{projectId}/change-requests", func(r chi.Router) {
		r.Post("/", h.create)
		r.Get("/", h.list)
	})
	r.Route("/change-requests/{changeId}", func(r chi.Router) {
		r.Get("/", h.get)
		r.Post("/decision", h.decide)
	})
}

// create 发起变更申请。
func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	projectID, err := idparse.FromURL(r, "projectId")
	if err != nil {
		response.Fail(w, errs.BadRequest(err.Error(), err))
		return
	}
	principal, queries, err := h.principal(r)
	if err != nil {
		response.Fail(w, err)
		return
	}
	if err := h.ensureProject(w, r, queries, projectID, principal); err != nil {
		return
	}
	var req CreateChangeRequestRequest
	if err := validate.Decode(r, &req); err != nil {
		response.Fail(w, errs.BadRequest(err.Error(), err))
		return
	}
	changeRequest, err := h.Service.Create(r.Context(), projectID, principal.ID, req)
	if err != nil {
		response.Fail(w, err)
		return
	}
	if err := h.recordAudit(r.Context(), r, principal, "change.create", changeRequest.ID, nil, changeRequest); err != nil {
		response.Fail(w, err)
		return
	}
	response.OK(w, CreateChangeRequestResponse{ChangeRequest: views.FromChangeRequest(changeRequest)})
}

// decide 客户确认或拒绝变更申请，批准时返回新基线。
func (h *Handler) decide(w http.ResponseWriter, r *http.Request) {
	changeID, err := idparse.FromURL(r, "changeId")
	if err != nil {
		response.Fail(w, errs.BadRequest(err.Error(), err))
		return
	}
	principal, queries, err := h.principal(r)
	if err != nil {
		response.Fail(w, err)
		return
	}
	changeRequest, err := h.Service.Get(r.Context(), changeID)
	if err != nil {
		response.Fail(w, err)
		return
	}
	project, err := queries.GetProject(r.Context(), changeRequest.ProjectID)
	if err != nil {
		response.Fail(w, errs.NotFound("项目不存在", err))
		return
	}
	if err := access.RequireSameOrganization(principal, project.OrganizationID); err != nil {
		response.Fail(w, err)
		return
	}
	if err := access.RequireOwner(principal); err != nil {
		response.Fail(w, err)
		return
	}
	var req DecisionRequest
	if err := validate.Decode(r, &req); err != nil {
		response.Fail(w, errs.BadRequest(err.Error(), err))
		return
	}
	decided, baseline, err := h.Service.Decide(r.Context(), changeID, principal.ID, req)
	if err != nil {
		response.Fail(w, err)
		return
	}
	if err := h.recordAudit(r.Context(), r, principal, "change.decision", changeID, changeRequest, decided); err != nil {
		response.Fail(w, err)
		return
	}
	response.OK(w, buildDecisionResponse(decided, baseline))
}

// ensureProject 校验项目归属，失败时写入错误响应。
func (h *Handler) ensureProject(w http.ResponseWriter, r *http.Request, queries *db.Queries, projectID uuid.UUID, principal access.Principal) error {
	project, err := queries.GetProject(r.Context(), projectID)
	if err != nil {
		response.Fail(w, errs.NotFound("项目不存在", err))
		return err
	}
	if err := access.RequireSameOrganization(principal, project.OrganizationID); err != nil {
		response.Fail(w, err)
		return err
	}
	return nil
}

// get 返回变更申请详情。
func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	changeID, err := idparse.FromURL(r, "changeId")
	if err != nil {
		response.Fail(w, errs.BadRequest(err.Error(), err))
		return
	}
	principal, queries, err := h.principal(r)
	if err != nil {
		response.Fail(w, err)
		return
	}
	changeRequest, err := h.Service.Get(r.Context(), changeID)
	if err != nil {
		response.Fail(w, err)
		return
	}
	if err := h.ensureProject(w, r, queries, changeRequest.ProjectID, principal); err != nil {
		return
	}
	response.OK(w, GetChangeRequestResponse{ChangeRequest: views.FromChangeRequest(changeRequest)})
}

// list 分页列出项目变更申请。
func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	projectID, err := idparse.FromURL(r, "projectId")
	if err != nil {
		response.Fail(w, errs.BadRequest(err.Error(), err))
		return
	}
	principal, queries, err := h.principal(r)
	if err != nil {
		response.Fail(w, err)
		return
	}
	if err := h.ensureProject(w, r, queries, projectID, principal); err != nil {
		return
	}
	pagination := response.ParsePagination(r)
	items, err := h.Service.List(r.Context(), projectID, int32(pagination.PageSize), int32((pagination.Page-1)*pagination.PageSize))
	if err != nil {
		response.Fail(w, err)
		return
	}
	total, err := h.Service.CountByProject(r.Context(), projectID)
	if err != nil {
		response.Fail(w, err)
		return
	}
	viewsList := make([]views.ChangeRequestView, 0, len(items))
	for _, item := range items {
		viewsList = append(viewsList, views.FromChangeRequest(item))
	}
	response.OKWithPage(w, ListChangeRequestsResponse{Items: viewsList}, response.PageOf(pagination, total))
}

// principal 解析当前访问主体与查询对象。
func (h *Handler) principal(r *http.Request) (access.Principal, *db.Queries, error) {
	queries := db.New(h.Service.Database)
	principal, err := access.FromRequest(r.Context(), r, queries)
	return principal, queries, err
}

// recordAudit 记录变更写操作的审计事件。
func (h *Handler) recordAudit(ctx context.Context, r *http.Request, principal access.Principal, action string, changeID uuid.UUID, before, after any) error {
	writer := audit.WriterFor(ctx, h.Service.Database)
	return writer.Record(ctx, audit.FromRequest(r, principal.ID), action, "change_request", changeID, before, after)
}

// buildDecisionResponse 构造变更决定响应，批准时附上新基线。
func buildDecisionResponse(decided db.ChangeRequest, baseline *db.ScopeBaseline) DecisionResponse {
	result := DecisionResponse{ChangeRequest: views.FromChangeRequest(decided)}
	if baseline != nil {
		view := views.FromBaseline(*baseline)
		result.Baseline = &view
	}
	return result
}
