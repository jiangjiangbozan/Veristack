// Package project 的 Handler 层，负责项目接口的数据绑定、权限校验与响应构造。
package project

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

// Handler 提供项目相关的 HTTP 接口。
type Handler struct {
	Service *Service // 项目业务服务
}

// Routes 注册项目模块的路由，全部需要鉴权。
func (h *Handler) Routes(r chi.Router) {
	r.Post("/projects", h.create)
	r.Get("/projects", h.list)
	r.Route("/projects/{projectId}", func(r chi.Router) {
		r.Get("/", h.get)
		r.Put("/", h.update)
		r.Delete("/", h.remove)
		r.Get("/overview", h.overview)
		r.Get("/members", h.listMembers)
		r.Post("/members", h.addMember)
		r.Delete("/members/{userId}", h.removeMember)
	})
}

// addMember 添加项目成员，仅管理员可操作。
func (h *Handler) addMember(w http.ResponseWriter, r *http.Request) {
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
	if err := access.RequireOwner(principal); err != nil {
		response.Fail(w, err)
		return
	}
	if err := h.ensureProject(w, r, queries, projectID, principal); err != nil {
		return
	}
	var req AddMemberRequest
	if err := validate.Decode(r, &req); err != nil {
		response.Fail(w, errs.BadRequest(err.Error(), err))
		return
	}
	user, err := h.Service.AddMember(r.Context(), projectID, req, principal.OrganizationID)
	if err != nil {
		response.Fail(w, err)
		return
	}
	response.OK(w, AddMemberResponse{Member: toMemberView(user.ID, user.DisplayName, user.Email, req.Role)})
}

// create 在当前用户所属组织下创建项目。
func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	principal, _, err := h.principal(r)
	if err != nil {
		response.Fail(w, err)
		return
	}
	var req CreateProjectRequest
	if err := validate.Decode(r, &req); err != nil {
		response.Fail(w, errs.BadRequest(err.Error(), err))
		return
	}
	project, err := h.Service.Create(r.Context(), principal.OrganizationID, principal.ID, req)
	if err != nil {
		response.Fail(w, err)
		return
	}
	if err := h.recordAudit(r.Context(), r, principal, "project.create", project.ID, nil, project); err != nil {
		response.Fail(w, err)
		return
	}
	response.OK(w, CreateProjectResponse{Project: views.FromProject(project)})
}

// ensureProject 读取项目并校验其组织归属，失败时直接写入错误响应。
func (h *Handler) ensureProject(w http.ResponseWriter, r *http.Request, queries *db.Queries, projectID uuid.UUID, principal access.Principal) error {
	project, err := h.Service.Get(r.Context(), projectID)
	if err != nil {
		response.Fail(w, err)
		return err
	}
	if err := access.RequireSameOrganization(principal, project.OrganizationID); err != nil {
		response.Fail(w, err)
		return err
	}
	return nil
}

// get 返回项目详情。
func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
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
	project, _ := h.Service.Get(r.Context(), projectID)
	response.OK(w, GetProjectResponse{Project: views.FromProject(project)})
}

// list 分页列出当前组织的项目。
func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	principal, _, err := h.principal(r)
	if err != nil {
		response.Fail(w, err)
		return
	}
	pagination := response.ParsePagination(r)
	projects, err := h.Service.List(r.Context(), principal.OrganizationID, int32(pagination.PageSize), int32((pagination.Page-1)*pagination.PageSize))
	if err != nil {
		response.Fail(w, err)
		return
	}
	total, err := h.Service.CountByOrganization(r.Context(), principal.OrganizationID)
	if err != nil {
		response.Fail(w, err)
		return
	}
	items := make([]views.ProjectView, 0, len(projects))
	for _, project := range projects {
		items = append(items, views.FromProject(project))
	}
	response.OKWithPage(w, ListProjectsResponse{Items: items}, response.PageOf(pagination, total))
}

// listMembers 列出项目成员。
func (h *Handler) listMembers(w http.ResponseWriter, r *http.Request) {
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
	rows, err := h.Service.ListMembers(r.Context(), projectID)
	if err != nil {
		response.Fail(w, err)
		return
	}
	items := make([]MemberView, 0, len(rows))
	for _, row := range rows {
		items = append(items, toMemberView(row.UserID, row.DisplayName, row.Email, row.Role))
	}
	response.OK(w, ListMembersResponse{Items: items})
}

// overview 返回项目概览统计。
func (h *Handler) overview(w http.ResponseWriter, r *http.Request) {
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
	result, err := h.Service.Overview(r.Context(), projectID)
	if err != nil {
		response.Fail(w, err)
		return
	}
	response.OK(w, result)
}

// principal 解析当前访问主体与查询对象。
func (h *Handler) principal(r *http.Request) (access.Principal, *db.Queries, error) {
	queries := db.New(h.Service.Database)
	principal, err := access.FromRequest(r.Context(), r, queries)
	return principal, queries, err
}

// recordAudit 记录项目写操作的审计事件。
func (h *Handler) recordAudit(ctx context.Context, r *http.Request, principal access.Principal, action string, projectID uuid.UUID, before, after any) error {
	writer := audit.WriterFor(ctx, h.Service.Database)
	return writer.Record(ctx, audit.FromRequest(r, principal.ID), action, "project", projectID, before, after)
}

// remove 软删除项目，仅管理员可操作。
func (h *Handler) remove(w http.ResponseWriter, r *http.Request) {
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
	if err := access.RequireOwner(principal); err != nil {
		response.Fail(w, err)
		return
	}
	if err := h.ensureProject(w, r, queries, projectID, principal); err != nil {
		return
	}
	project, _ := h.Service.Get(r.Context(), projectID)
	if err := h.Service.SoftDelete(r.Context(), projectID); err != nil {
		response.Fail(w, err)
		return
	}
	if err := h.recordAudit(r.Context(), r, principal, "project.delete", projectID, project, nil); err != nil {
		response.Fail(w, err)
		return
	}
	response.OK(w, map[string]bool{"deleted": true})
}

// removeMember 移除项目成员，仅管理员可操作。
func (h *Handler) removeMember(w http.ResponseWriter, r *http.Request) {
	projectID, err := idparse.FromURL(r, "projectId")
	if err != nil {
		response.Fail(w, errs.BadRequest(err.Error(), err))
		return
	}
	userID, err := idparse.FromURL(r, "userId")
	if err != nil {
		response.Fail(w, errs.BadRequest(err.Error(), err))
		return
	}
	principal, queries, err := h.principal(r)
	if err != nil {
		response.Fail(w, err)
		return
	}
	if err := access.RequireOwner(principal); err != nil {
		response.Fail(w, err)
		return
	}
	if err := h.ensureProject(w, r, queries, projectID, principal); err != nil {
		return
	}
	if err := h.Service.RemoveMember(r.Context(), projectID, userID); err != nil {
		response.Fail(w, err)
		return
	}
	response.OK(w, map[string]bool{"deleted": true})
}

// update 更新项目信息。
func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
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
	var req UpdateProjectRequest
	if err := validate.Decode(r, &req); err != nil {
		response.Fail(w, errs.BadRequest(err.Error(), err))
		return
	}
	project, err := h.Service.Update(r.Context(), projectID, req)
	if err != nil {
		response.Fail(w, err)
		return
	}
	if err := h.recordAudit(r.Context(), r, principal, "project.update", projectID, nil, project); err != nil {
		response.Fail(w, err)
		return
	}
	response.OK(w, UpdateProjectResponse{Project: views.FromProject(project)})
}

// toMemberView 把成员信息转换为响应视图。
func toMemberView(id uuid.UUID, displayName, email, role string) MemberView {
	return MemberView{DisplayName: displayName, Email: email, ID: id, Role: role}
}
