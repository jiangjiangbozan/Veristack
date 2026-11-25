// Package identity 的 Handler 层，负责认证接口的数据绑定、校验、权限与响应构造。
package identity

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"api/internal/platform/errs"
	"api/internal/platform/middleware"
	"api/internal/platform/response"
	"api/internal/platform/validate"
	"api/internal/platform/views"
)

// Handler 提供认证相关的 HTTP 接口。
type Handler struct {
	Service *Service // 身份业务服务
}

// Routes 注册身份模块的路由，register 与 login 为公开接口。
func (h *Handler) Routes(r chi.Router) {
	r.Post("/auth/register", h.register)
	r.Post("/auth/login", h.login)
}

// MeRoutes 注册需要鉴权的当前用户接口。
func (h *Handler) MeRoutes(r chi.Router) {
	r.Get("/auth/me", h.me)
}

// login 处理登录请求，成功时返回令牌与用户信息。
func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := validate.Decode(r, &req); err != nil {
		response.Fail(w, errs.BadRequest(err.Error(), err))
		return
	}
	user, token, err := h.Service.Login(r.Context(), req)
	if err != nil {
		response.Fail(w, err)
		return
	}
	response.OK(w, LoginResponse{Token: token, User: views.FromUser(user)})
}

// me 返回当前登录用户的资料。
func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	claims := middleware.UserFrom(r)
	if claims == nil {
		response.Fail(w, errs.Unauthorized("未认证", nil))
		return
	}
	actorID, err := uuid.Parse(claims.UserID)
	if err != nil {
		response.Fail(w, errs.Unauthorized("用户标识无效", err))
		return
	}
	user, err := h.Service.GetUser(r.Context(), actorID)
	if err != nil {
		response.Fail(w, err)
		return
	}
	response.OK(w, MeResponse{User: views.FromUser(user)})
}

// register 处理组织注册，创建组织与 owner 用户。
func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := validate.Decode(r, &req); err != nil {
		response.Fail(w, errs.BadRequest(err.Error(), err))
		return
	}
	user, token, err := h.Service.Register(r.Context(), req)
	if err != nil {
		response.Fail(w, err)
		return
	}
	response.OK(w, RegisterResponse{Token: token, User: views.FromUser(user)})
}
