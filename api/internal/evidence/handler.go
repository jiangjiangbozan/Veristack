// Package evidence 的 Handler 层，负责证据接口的数据绑定、权限校验与响应构造。
package evidence

import (
	"io"
	"net/http"
	"strings"

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

// Handler 提供证据相关的 HTTP 接口。
type Handler struct {
	Service *Service // 证据业务服务
}

const (
	// uploadMemoryLimit 是表单解析时保存在内存中的上限，超出部分落临时文件。
	uploadMemoryLimit = 8 << 20
)

// Routes 注册证据模块路由，全部需要鉴权。
func (h *Handler) Routes(r chi.Router) {
	r.Route("/deliveries/{deliveryId}/evidence", func(r chi.Router) {
		r.Post("/", h.upload)
		r.Get("/", h.list)
	})
	r.Get("/projects/{projectId}/evidence", h.listByProject)
	r.Get("/evidence/{evidenceId}/file", h.download)
}

// listByProject 列出项目下全部证据，便于从项目视角查看自动化采集结果。
func (h *Handler) listByProject(w http.ResponseWriter, r *http.Request) {
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
	project, err := queries.GetProject(r.Context(), projectID)
	if err != nil {
		response.Fail(w, errs.NotFound("项目不存在", err))
		return
	}
	if err := access.RequireSameOrganization(principal, project.OrganizationID); err != nil {
		response.Fail(w, err)
		return
	}
	items, err := h.Service.ListByProject(r.Context(), projectID)
	if err != nil {
		response.Fail(w, err)
		return
	}
	evidenceViews := make([]views.EvidenceView, 0, len(items))
	for _, item := range items {
		evidenceViews = append(evidenceViews, views.FromEvidence(item))
	}
	response.OK(w, ListProjectEvidenceResponse{Items: evidenceViews})
}

// createManual 登记一条人工说明型证据，不涉及附件上传。
func (h *Handler) createManual(w http.ResponseWriter, r *http.Request, delivery db.Delivery, principal access.Principal) {
	var req CreateEvidenceRequest
	if err := validate.Decode(r, &req); err != nil {
		response.Fail(w, errs.BadRequest(err.Error(), err))
		return
	}
	item, err := h.Service.CreateManual(r.Context(), delivery.ProjectID, delivery.ID, req)
	if err != nil {
		response.Fail(w, err)
		return
	}
	h.recordAudit(r, principal, "evidence.create", item.ID, nil, item)
	response.OK(w, CreateEvidenceResponse{Evidence: views.FromEvidence(item)})
}

// download 下载证据附件。
func (h *Handler) download(w http.ResponseWriter, r *http.Request) {
	evidenceID, err := idparse.FromURL(r, "evidenceId")
	if err != nil {
		response.Fail(w, errs.BadRequest(err.Error(), err))
		return
	}
	principal, _, err := h.principal(r)
	if err != nil {
		response.Fail(w, err)
		return
	}
	item, err := h.Service.Get(r.Context(), evidenceID)
	if err != nil {
		response.Fail(w, err)
		return
	}
	deliveryID, err := idparse.FromUUID(item.DeliveryID)
	if err != nil {
		response.Fail(w, errs.NotFound("证据未关联交付版本", err))
		return
	}
	if _, err := h.ensureDelivery(w, r, db.New(h.Service.Database), deliveryID, principal); err != nil {
		return
	}
	content, err := h.Service.Open(r.Context(), item.FileKey.String)
	if err != nil {
		response.Fail(w, err)
		return
	}
	defer content.Close()
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+item.Title+"\"")
	if _, err := io.Copy(w, content); err != nil {
		response.Fail(w, errs.Internal(err))
	}
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

// list 列出交付版本的证据时间线。
func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
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
	items, err := h.Service.ListByDelivery(r.Context(), deliveryID)
	if err != nil {
		response.Fail(w, err)
		return
	}
	viewsList := make([]views.EvidenceView, 0, len(items))
	for _, item := range items {
		viewsList = append(viewsList, views.FromEvidence(item))
	}
	response.OK(w, ListEvidenceResponse{Items: viewsList})
}

// principal 解析当前访问主体与查询对象。
func (h *Handler) principal(r *http.Request) (access.Principal, *db.Queries, error) {
	queries := db.New(h.Service.Database)
	principal, err := access.FromRequest(r.Context(), r, queries)
	return principal, queries, err
}

// recordAudit 记录证据写操作的审计事件。
func (h *Handler) recordAudit(r *http.Request, principal access.Principal, action string, evidenceID uuid.UUID, before, after any) {
	writer := audit.Writer{Queries: db.New(h.Service.Database)}
	_ = writer.Record(r.Context(), audit.FromRequest(r, principal.ID), action, "evidence_item", evidenceID, before, after)
}

// upload 接收证据登记请求，按内容类型区分附件上传与人工说明。
func (h *Handler) upload(w http.ResponseWriter, r *http.Request) {
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
	if isMultipartRequest(r) {
		h.uploadWithFile(w, r, delivery, principal)
		return
	}
	h.createManual(w, r, delivery, principal)
}

// uploadWithFile 处理带附件的证据上传请求。
func (h *Handler) uploadWithFile(w http.ResponseWriter, r *http.Request, delivery db.Delivery, principal access.Principal) {
	if err := r.ParseMultipartForm(uploadMemoryLimit); err != nil {
		response.Fail(w, errs.BadRequest("上传内容格式无效", err))
		return
	}
	var req UploadRequest
	if err := validate.DecodeForm(r, &req); err != nil {
		response.Fail(w, errs.BadRequest(err.Error(), err))
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		response.Fail(w, errs.BadRequest("缺少上传文件", err))
		return
	}
	defer file.Close()
	item, err := h.Service.CreateUpload(r.Context(), delivery.ProjectID, delivery.ID, req, file, header)
	if err != nil {
		response.Fail(w, err)
		return
	}
	h.recordAudit(r, principal, "evidence.upload", item.ID, nil, item)
	response.OK(w, CreateEvidenceResponse{Evidence: views.FromEvidence(item)})
}

// isMultipartRequest 判断请求是否为带附件的表单上传。
func isMultipartRequest(r *http.Request) bool {
	return strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data")
}
