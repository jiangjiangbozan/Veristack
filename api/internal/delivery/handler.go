// Package delivery 的 Handler 层，负责交付接口的数据绑定、权限校验与响应构造。
package delivery

import (
	"crypto/ed25519"
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

// Handler 提供交付版本相关的 HTTP 接口。
type Handler struct {
	Service    *Service           // 交付业务服务
	SigningKey ed25519.PrivateKey // 交付清单签名私钥，未配置时导出的 Pack 不带签名
}

// Routes 注册交付模块路由，全部需要鉴权。
func (h *Handler) Routes(r chi.Router) {
	r.Route("/projects/{projectId}/deliveries", func(r chi.Router) {
		r.Post("/", h.create)
		r.Get("/", h.list)
	})
	r.Route("/deliveries/{deliveryId}", func(r chi.Router) {
		r.Get("/", h.get)
		r.Get("/manifest", h.getManifest)
		r.Post("/submit", h.submit)
		r.Post("/pack", h.exportPack)
	})
}

// exportPack 导出 Evidence Pack，供客户或第三方独立核验交付内容。
func (h *Handler) exportPack(w http.ResponseWriter, r *http.Request) {
	deliveryID, err := idparse.FromURL(r, "deliveryId")
	if err != nil {
		response.Fail(w, errs.BadRequest(err.Error(), err))
		return
	}
	principal, _, err := h.principal(r)
	if err != nil {
		response.Fail(w, err)
		return
	}
	delivery, err := h.Service.Get(r.Context(), deliveryID)
	if err != nil {
		response.Fail(w, err)
		return
	}
	if err := h.ensureDelivery(w, r, delivery, principal); err != nil {
		return
	}
	root, manifestSHA, signed, err := h.Service.ExportPack(r.Context(), deliveryID, h.SigningKey)
	if err != nil {
		response.Fail(w, err)
		return
	}
	response.OK(w, PackResponse{
		DownloadURL: "/api/v1/deliveries/" + deliveryID.String() + "/pack/download",
		ManifestSHA: manifestSHA,
		PackPath:    root,
		Signed:      signed,
	})
}

// create 创建交付版本并冻结交付清单。
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
	var req CreateDeliveryRequest
	if err := validate.Decode(r, &req); err != nil {
		response.Fail(w, errs.BadRequest(err.Error(), err))
		return
	}
	delivery, err := h.Service.Create(r.Context(), projectID, req)
	if err != nil {
		response.Fail(w, err)
		return
	}
	h.recordAudit(r, principal, "delivery.create", delivery.ID, nil, delivery)
	response.OK(w, CreateDeliveryResponse{Delivery: views.FromDelivery(delivery)})
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

// get 返回交付版本详情。
func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	deliveryID, err := idparse.FromURL(r, "deliveryId")
	if err != nil {
		response.Fail(w, errs.BadRequest(err.Error(), err))
		return
	}
	principal, _, err := h.principal(r)
	if err != nil {
		response.Fail(w, err)
		return
	}
	delivery, err := h.Service.Get(r.Context(), deliveryID)
	if err != nil {
		response.Fail(w, err)
		return
	}
	if err := h.ensureDelivery(w, r, delivery, principal); err != nil {
		return
	}
	response.OK(w, GetDeliveryResponse{Delivery: views.FromDelivery(delivery)})
}

// ensureDelivery 校验交付归属，失败时写入错误响应。
func (h *Handler) ensureDelivery(w http.ResponseWriter, r *http.Request, delivery db.Delivery, principal access.Principal) error {
	project, err := db.New(h.Service.Database).GetProject(r.Context(), delivery.ProjectID)
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

// getManifest 返回冻结的交付清单快照。
func (h *Handler) getManifest(w http.ResponseWriter, r *http.Request) {
	deliveryID, err := idparse.FromURL(r, "deliveryId")
	if err != nil {
		response.Fail(w, errs.BadRequest(err.Error(), err))
		return
	}
	principal, _, err := h.principal(r)
	if err != nil {
		response.Fail(w, err)
		return
	}
	delivery, err := h.Service.Get(r.Context(), deliveryID)
	if err != nil {
		response.Fail(w, err)
		return
	}
	if err := h.ensureDelivery(w, r, delivery, principal); err != nil {
		return
	}
	manifest, err := h.Service.GetManifest(r.Context(), deliveryID)
	if err != nil {
		response.Fail(w, err)
		return
	}
	response.OK(w, manifest)
}

// list 分页列出项目交付版本。
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
	viewsList := make([]views.DeliveryView, 0, len(items))
	for _, item := range items {
		viewsList = append(viewsList, views.FromDelivery(item))
	}
	response.OKWithPage(w, ListDeliveriesResponse{Items: viewsList}, response.PageOf(pagination, total))
}

// principal 解析当前访问主体与查询对象。
func (h *Handler) principal(r *http.Request) (access.Principal, *db.Queries, error) {
	queries := db.New(h.Service.Database)
	principal, err := access.FromRequest(r.Context(), r, queries)
	return principal, queries, err
}

// recordAudit 记录交付写操作的审计事件。
func (h *Handler) recordAudit(r *http.Request, principal access.Principal, action string, deliveryID uuid.UUID, before, after any) {
	writer := audit.Writer{Queries: db.New(h.Service.Database)}
	_ = writer.Record(r.Context(), audit.FromRequest(r, principal.ID), action, "delivery", deliveryID, before, after)
}

// submit 把草稿交付提交为待验收状态。
func (h *Handler) submit(w http.ResponseWriter, r *http.Request) {
	deliveryID, err := idparse.FromURL(r, "deliveryId")
	if err != nil {
		response.Fail(w, errs.BadRequest(err.Error(), err))
		return
	}
	principal, _, err := h.principal(r)
	if err != nil {
		response.Fail(w, err)
		return
	}
	before, err := h.Service.Get(r.Context(), deliveryID)
	if err != nil {
		response.Fail(w, err)
		return
	}
	if err := h.ensureDelivery(w, r, before, principal); err != nil {
		return
	}
	delivery, err := h.Service.Submit(r.Context(), deliveryID)
	if err != nil {
		response.Fail(w, err)
		return
	}
	h.recordAudit(r, principal, "delivery.submit", deliveryID, before, delivery)
	response.OK(w, SubmitDeliveryResponse{Delivery: views.FromDelivery(delivery)})
}
