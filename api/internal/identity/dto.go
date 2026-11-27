// Package identity 的 DTO 文件，定义认证与用户接口的请求和响应结构。
package identity

import (
	"api/internal/platform/views"
)

// RegisterRequest 是注册组织的请求体，同时创建组织与 owner 用户。
type RegisterRequest struct {
	DisplayName      string `json:"displayName" validate:"required,max=50"`
	Email            string `json:"email" validate:"required,email,max=120"`
	OrganizationName string `json:"organizationName" validate:"required,max=80"`
	Password         string `json:"password" validate:"required,min=8,max=128"`
}

// RegisterResponse 是注册成功响应，直接返回令牌供前端登录。
type RegisterResponse struct {
	Token string         `json:"token"`
	User  views.UserView `json:"user"`
}

// LoginRequest 是登录请求体。
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// LoginResponse 是登录成功响应。
type LoginResponse struct {
	Token string         `json:"token"`
	User  views.UserView `json:"user"`
}

// MeResponse 是当前用户查询响应。
type MeResponse struct {
	User views.UserView `json:"user"`
}

// ChangePasswordRequest 是修改密码请求体。
type ChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword" validate:"required"`
	NewPassword     string `json:"newPassword" validate:"required,min=8,max=128"`
}

// ForgotPasswordRequest 是申请密码重置邮件的请求体。
type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// ResetPasswordRequest 是使用一次性令牌重置密码的请求体。
type ResetPasswordRequest struct {
	NewPassword string `json:"newPassword" validate:"required,min=8,max=128"`
	Token       string `json:"token" validate:"required"`
}

// VerifyEmailRequest 是提交邮箱验证码的请求体。
type VerifyEmailRequest struct {
	Code string `json:"code" validate:"required,len=6"`
}
