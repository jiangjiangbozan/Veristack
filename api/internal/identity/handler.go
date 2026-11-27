// Package identity 的 Handler 层，负责认证接口的数据绑定、校验、权限与响应构造。
package identity

import (
	"net"
	"net/http"
	"strings"

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
	Cooldown *CooldownLimiter // 邮件等副作用冷却限流器
	Limiter  *LoginLimiter    // 登录限流器
	Service  *Service         // 身份业务服务
}

// Routes 注册身份模块的路由，register、login、找回密码为公开接口。
func (h *Handler) Routes(r chi.Router) {
	r.Post("/auth/register", h.register)
	r.Post("/auth/login", h.login)
	r.Post("/auth/password/forgot", h.forgotPassword)
	r.Post("/auth/password/reset", h.resetPassword)
}

// MeRoutes 注册需要鉴权的当前用户与账户安全接口。
func (h *Handler) MeRoutes(r chi.Router) {
	r.Get("/auth/me", h.me)
	r.Post("/auth/logout", h.logout)
	r.Post("/auth/password", h.changePassword)
	r.Post("/auth/email/verification-code", h.requestEmailVerificationCode)
	r.Post("/auth/email/verify", h.verifyEmail)
}

// login 处理登录请求，成功时返回令牌与用户信息。
func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := validate.Decode(r, &req); err != nil {
		response.Fail(w, errs.BadRequest(err.Error(), err))
		return
	}
	key := loginAttemptKey(r, req.Email)
	if !h.Limiter.Allow(key) {
		response.Fail(w, errs.TooManyRequests("登录尝试过于频繁，请稍后再试", nil))
		return
	}
	user, token, err := h.Service.Login(r.Context(), req)
	if err != nil {
		h.Limiter.Fail(key)
		response.Fail(w, err)
		return
	}
	h.Limiter.Reset(key)
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

// changePassword 修改当前用户密码，成功后旧会话令牌全部失效。
func (h *Handler) changePassword(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUserID(w, r)
	if !ok {
		return
	}
	var req ChangePasswordRequest
	if err := validate.Decode(r, &req); err != nil {
		response.Fail(w, errs.BadRequest(err.Error(), err))
		return
	}
	if err := h.Service.ChangePassword(r.Context(), userID, req.CurrentPassword, req.NewPassword); err != nil {
		response.Fail(w, err)
		return
	}
	response.OK(w, map[string]bool{"changed": true})
}

// forgotPassword 申请密码重置邮件；无论邮箱是否存在都返回成功。
func (h *Handler) forgotPassword(w http.ResponseWriter, r *http.Request) {
	var req ForgotPasswordRequest
	if err := validate.Decode(r, &req); err != nil {
		response.Fail(w, errs.BadRequest(err.Error(), err))
		return
	}
	if !h.cooldownAllows("password-forgot|" + loginAttemptKey(r, req.Email)) {
		response.Fail(w, errs.TooManyRequests("请求过于频繁，请稍后再试", nil))
		return
	}
	if err := h.Service.RequestPasswordReset(r.Context(), req.Email); err != nil {
		response.Fail(w, err)
		return
	}
	response.OK(w, map[string]bool{"sent": true})
}

// logout 使当前用户已签发的会话令牌全部失效。
func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUserID(w, r)
	if !ok {
		return
	}
	if err := h.Service.Logout(r.Context(), userID); err != nil {
		response.Fail(w, err)
		return
	}
	response.OK(w, map[string]bool{"loggedOut": true})
}

// requestEmailVerificationCode 为当前用户邮箱发送验证码。
func (h *Handler) requestEmailVerificationCode(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUserID(w, r)
	if !ok {
		return
	}
	if !h.cooldownAllows("email-verification|" + userID.String()) {
		response.Fail(w, errs.TooManyRequests("验证码发送过于频繁，请稍后再试", nil))
		return
	}
	if err := h.Service.RequestEmailVerificationCode(r.Context(), userID); err != nil {
		response.Fail(w, err)
		return
	}
	response.OK(w, map[string]bool{"sent": true})
}

// resetPassword 使用一次性令牌设置新密码。
func (h *Handler) resetPassword(w http.ResponseWriter, r *http.Request) {
	var req ResetPasswordRequest
	if err := validate.Decode(r, &req); err != nil {
		response.Fail(w, errs.BadRequest(err.Error(), err))
		return
	}
	if err := h.Service.ResetPassword(r.Context(), req.Token, req.NewPassword); err != nil {
		response.Fail(w, err)
		return
	}
	response.OK(w, map[string]bool{"reset": true})
}

// verifyEmail 提交邮箱验证码并返回更新后的用户。
func (h *Handler) verifyEmail(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUserID(w, r)
	if !ok {
		return
	}
	var req VerifyEmailRequest
	if err := validate.Decode(r, &req); err != nil {
		response.Fail(w, errs.BadRequest(err.Error(), err))
		return
	}
	user, err := h.Service.VerifyEmail(r.Context(), userID, req.Code)
	if err != nil {
		response.Fail(w, err)
		return
	}
	response.OK(w, MeResponse{User: views.FromUser(user)})
}

// currentUserID 从请求上下文解析当前用户 ID，失败时写入错误响应。
func (h *Handler) currentUserID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
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

// cooldownAllows 判断冷却限流器是否放行；未注入时默认放行，便于独立测试 Handler。
func (h *Handler) cooldownAllows(key string) bool {
	return h.Cooldown == nil || h.Cooldown.Allow(key)
}

// loginAttemptKey 组合来源 IP 与邮箱，避免单一维度被绕过。
//
// 只使用连接来源地址，不信任客户端可伪造的 X-Forwarded-For；
// 部署在反向代理后时，所有请求共享代理地址，但邮箱维度仍能限制单账号爆破。
func loginAttemptKey(r *http.Request, email string) string {
	host := r.RemoteAddr
	if parsed, _, err := net.SplitHostPort(host); err == nil {
		host = parsed
	}
	return host + "|" + strings.ToLower(strings.TrimSpace(email))
}
