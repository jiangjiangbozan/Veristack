// Package scope 的 Handler 层，负责基线接口的数据绑定、权限校验与响应构造。
package scope

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

// Handler 提供范围基线相关的 HTTP 接口。
type Handler struct {
	Service *Service // 基线业务服务
}

// Routes 注册基线模块的路由，全部需要鉴权。
func (h *Handler) Routes(r chi.Router) {
	r.Route("/projects/{projectId}/scope-baselines", func(r chi.Router) {
		r.Post("/", h.create)
		r.Get("/", h.list)
	})
	r.Route("/scope-baselines/{baselineId}", func(r chi.Router) {
		r.Get("/", h.get)
		r.Get("/diff", h.diff)
	})
}

// create 创建并冻结新的需求基线。
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
	var req CreateBaselineRequest
	if err := validate.Decode(r, &req); err != nil {
		response.Fail(w, errs.BadRequest(err.Error(), err))
		return
	}
	baseline, err := h.Service.Create(r.Context(), projectID, req)
	if err != nil {
		response.Fail(w, err)
		return
	}
	if err := h.recordAudit(r.Context(), r, principal, "scope.baseline.create", baseline.ID, nil, baseline); err != nil {
		response.Fail(w, err)
		return
	}
	response.OK(w, CreateBaselineResponse{Baseline: views.FromBaseline(baseline)})
}

// diff 对比两个基线的需求差异。
func (h *Handler) diff(w http.ResponseWriter, r *http.Request) {
	baseID, err := idparse.FromURL(r, "baselineId")
	if err != nil {
		response.Fail(w, errs.BadRequest(err.Error(), err))
		return
	}
	targetID, err := uuid.Parse(r.URL.Query().Get("against"))
	if err != nil {
		response.Fail(w, errs.BadRequest("对比基线 ID 无效", err))
		return
	}
	principal, _, err := h.principal(r)
	if err != nil {
		response.Fail(w, err)
		return
	}
	base, err := h.Service.Get(r.Context(), baseID)
	if err != nil {
		response.Fail(w, err)
		return
	}
	if err := h.ensureProject(w, r, db.New(h.Service.Database), base.ProjectID, principal); err != nil {
		return
	}
	result, err := h.Service.Diff(r.Context(), baseID, targetID)
	if err != nil {
		response.Fail(w, err)
		return
	}
	response.OK(w, result)
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

// get 返回基线详情。
func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	baselineID, err := idparse.FromURL(r, "baselineId")
	if err != nil {
		response.Fail(w, errs.BadRequest(err.Error(), err))
		return
	}
	principal, _, err := h.principal(r)
	if err != nil {
		response.Fail(w, err)
		return
	}
	baseline, err := h.Service.Get(r.Context(), baselineID)
	if err != nil {
		response.Fail(w, err)
		return
	}
	if err := h.ensureProject(w, r, db.New(h.Service.Database), baseline.ProjectID, principal); err != nil {
		return
	}
	response.OK(w, GetBaselineResponse{Baseline: views.FromBaseline(baseline)})
}

// list 列出项目全部基线版本。
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
	baselines, err := h.Service.List(r.Context(), projectID)
	if err != nil {
		response.Fail(w, err)
		return
	}
	items := make([]views.BaselineView, 0, len(baselines))
	for _, baseline := range baselines {
		items = append(items, views.FromBaseline(baseline))
	}
	response.OK(w, ListBaselinesResponse{Items: items})
}

// principal 解析当前访问主体与查询对象。
func (h *Handler) principal(r *http.Request) (access.Principal, *db.Queries, error) {
	queries := db.New(h.Service.Database)
	principal, err := access.FromRequest(r.Context(), r, queries)
	return principal, queries, err
}

// recordAudit 记录基线写操作的审计事件。
func (h *Handler) recordAudit(ctx context.Context, r *http.Request, principal access.Principal, action string, baselineID uuid.UUID, before, after any) error {
	writer := audit.WriterFor(ctx, h.Service.Database)
	return writer.Record(ctx, audit.FromRequest(r, principal.ID), action, "scope_baseline", baselineID, before, after)
}
