// Package notification 的 Handler 层，提供站内通知查询与已读接口。
package notification

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"api/internal/platform/dbutil"
	"api/internal/platform/errs"
	"api/internal/platform/middleware"
	"api/internal/platform/response"
	"api/internal/platform/views"
)

// Handler 提供通知相关的 HTTP 接口。
type Handler struct {
	Service *Service // 通知业务服务
}

// Routes 注册需要登录的通知接口。
func (h *Handler) Routes(r chi.Router) {
	r.Get("/notifications", h.list)
	r.Get("/notifications/unread-count", h.unreadCount)
	r.Post("/notifications/read-all", h.markAllRead)
	r.Post("/notifications/{notificationId}/read", h.markRead)
}

// list 返回当前用户的站内通知。
func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	recipientID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	pagination := response.ParsePagination(r)
	items, err := h.Service.List(r.Context(), recipientID, int32(pagination.PageSize), int32((pagination.Page-1)*pagination.PageSize))
	if err != nil {
		response.Fail(w, errs.Internal(err))
		return
	}
	result := make([]views.NotificationView, 0, len(items))
	for _, item := range items {
		result = append(result, views.NotificationView{
			Body:      item.Body,
			CreatedAt: dbutil.TimeString(item.CreatedAt),
			ID:        item.ID,
			Link:      item.Link,
			ReadAt:    dbutil.NullableTimeString(item.ReadAt),
			Title:     item.Title,
			Type:      item.Type,
		})
	}
	response.OK(w, result)
}

// unreadCount 返回当前用户的未读通知数量。
func (h *Handler) unreadCount(w http.ResponseWriter, r *http.Request) {
	recipientID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	count, err := h.Service.UnreadCount(r.Context(), recipientID)
	if err != nil {
		response.Fail(w, errs.Internal(err))
		return
	}
	response.OK(w, map[string]int64{"count": count})
}

// markRead 把一条通知标记为已读。
func (h *Handler) markRead(w http.ResponseWriter, r *http.Request) {
	recipientID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	notificationID, err := uuid.Parse(chi.URLParam(r, "notificationId"))
	if err != nil {
		response.Fail(w, errs.BadRequest("通知标识无效", err))
		return
	}
	if err := h.Service.MarkRead(r.Context(), recipientID, notificationID); err != nil {
		response.Fail(w, errs.Internal(err))
		return
	}
	response.OK(w, map[string]bool{"read": true})
}

// markAllRead 把当前用户的全部未读通知标记为已读。
func (h *Handler) markAllRead(w http.ResponseWriter, r *http.Request) {
	recipientID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	count, err := h.Service.MarkAllRead(r.Context(), recipientID)
	if err != nil {
		response.Fail(w, errs.Internal(err))
		return
	}
	response.OK(w, map[string]int64{"updated": count})
}

// currentUserID 从认证上下文提取当前用户 ID。
func currentUserID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	claims := middleware.UserFrom(r)
	if claims == nil {
		response.Fail(w, errs.Unauthorized("未认证", nil))
		return uuid.Nil, false
	}
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		response.Fail(w, errs.Unauthorized("用户标识无效", err))
		return uuid.Nil, false
	}
	return userID, true
}
