// Package milestone 的 Handler 层，负责里程碑接口的数据绑定、权限校验与响应构造。
package milestone

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

// Handler 提供里程碑相关的 HTTP 接口。
type Handler struct {
	Service *Service // 里程碑业务服务
}

// Routes 注册里程碑模块路由。
func (h *Handler) Routes(r chi.Router) {
	r.Route("/projects/{projectId}/milestones", func(r chi.Router) {
		r.Post("/", h.create)
		r.Get("/", h.list)
	})
	r.Route("/milestones/{milestoneId}", func(r chi.Router) {
		r.Get("/", h.get)
		r.Put("/", h.update)
		r.Delete("/", h.remove)
	})
}

// create 在项目下创建里程碑。
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
	var req CreateMilestoneRequest
	if err := validate.Decode(r, &req); err != nil {
		response.Fail(w, errs.BadRequest(err.Error(), err))
		return
	}
	milestone, err := h.Service.Create(r.Context(), projectID, req)
	if err != nil {
		response.Fail(w, err)
		return
	}
	if err := h.recordAudit(r.Context(), r, principal, "milestone.create", milestone.ID, nil, milestone); err != nil {
		response.Fail(w, err)
		return
	}
	response.OK(w, CreateMilestoneResponse{Milestone: views.FromMilestone(milestone)})
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

// get 返回里程碑详情。
func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	milestoneID, err := idparse.FromURL(r, "milestoneId")
	if err != nil {
		response.Fail(w, errs.BadRequest(err.Error(), err))
		return
	}
	principal, queries, err := h.principal(r)
	if err != nil {
		response.Fail(w, err)
		return
	}
	milestone, err := h.Service.Get(r.Context(), milestoneID)
	if err != nil {
		response.Fail(w, err)
		return
	}
	if err := h.ensureProject(w, r, queries, milestone.ProjectID, principal); err != nil {
		return
	}
	response.OK(w, GetMilestoneResponse{Milestone: views.FromMilestone(milestone)})
}

// list 列出项目里程碑。
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
	milestones, err := h.Service.List(r.Context(), projectID)
	if err != nil {
		response.Fail(w, err)
		return
	}
	items := make([]views.MilestoneView, 0, len(milestones))
	for _, milestone := range milestones {
		items = append(items, views.FromMilestone(milestone))
	}
	response.OK(w, ListMilestonesResponse{Items: items})
}

// principal 解析当前访问主体与查询对象。
func (h *Handler) principal(r *http.Request) (access.Principal, *db.Queries, error) {
	queries := db.New(h.Service.Database)
	principal, err := access.FromRequest(r.Context(), r, queries)
	return principal, queries, err
}

// recordAudit 记录里程碑写操作的审计事件。
func (h *Handler) recordAudit(ctx context.Context, r *http.Request, principal access.Principal, action string, milestoneID uuid.UUID, before, after any) error {
	writer := audit.WriterFor(ctx, h.Service.Database)
	return writer.Record(ctx, audit.FromRequest(r, principal.ID), action, "milestone", milestoneID, before, after)
}

// remove 软删除里程碑。
func (h *Handler) remove(w http.ResponseWriter, r *http.Request) {
	milestoneID, err := idparse.FromURL(r, "milestoneId")
	if err != nil {
		response.Fail(w, errs.BadRequest(err.Error(), err))
		return
	}
	principal, queries, err := h.principal(r)
	if err != nil {
		response.Fail(w, err)
		return
	}
	milestone, err := h.Service.Get(r.Context(), milestoneID)
	if err != nil {
		response.Fail(w, err)
		return
	}
	if err := h.ensureProject(w, r, queries, milestone.ProjectID, principal); err != nil {
		return
	}
	if err := h.Service.SoftDelete(r.Context(), milestoneID); err != nil {
		response.Fail(w, err)
		return
	}
	if err := h.recordAudit(r.Context(), r, principal, "milestone.delete", milestoneID, milestone, nil); err != nil {
		response.Fail(w, err)
		return
	}
	response.OK(w, map[string]bool{"deleted": true})
}

// update 更新里程碑。
func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	milestoneID, err := idparse.FromURL(r, "milestoneId")
	if err != nil {
		response.Fail(w, errs.BadRequest(err.Error(), err))
		return
	}
	principal, queries, err := h.principal(r)
	if err != nil {
		response.Fail(w, err)
		return
	}
	milestone, err := h.Service.Get(r.Context(), milestoneID)
	if err != nil {
		response.Fail(w, err)
		return
	}
	if err := h.ensureProject(w, r, queries, milestone.ProjectID, principal); err != nil {
		return
	}
	var req UpdateMilestoneRequest
	if err := validate.Decode(r, &req); err != nil {
		response.Fail(w, errs.BadRequest(err.Error(), err))
		return
	}
	updated, err := h.Service.Update(r.Context(), milestoneID, req)
	if err != nil {
		response.Fail(w, err)
		return
	}
	if err := h.recordAudit(r.Context(), r, principal, "milestone.update", milestoneID, milestone, updated); err != nil {
		response.Fail(w, err)
		return
	}
	response.OK(w, CreateMilestoneResponse{Milestone: views.FromMilestone(updated)})
}
