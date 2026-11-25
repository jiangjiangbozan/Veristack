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
