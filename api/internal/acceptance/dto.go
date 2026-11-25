// Package acceptance 的 DTO 文件，定义客户验收与异议闭环接口的请求和响应结构。
package acceptance

import (
	"api/internal/platform/views"
)

// CreateShareTokenRequest 是生成客户分享链接的请求体。
type CreateShareTokenRequest struct {
	ExpiresInDays int `json:"expiresInDays" validate:"omitempty,min=1,max=90"`
}

// CreateShareTokenResponse 是分享链接响应，明文令牌只在创建时返回一次。
type CreateShareTokenResponse struct {
	ExpiresAt string `json:"expiresAt"` // 过期时间
	Token     string `json:"token"`     // 明文令牌
	URL       string `json:"url"`       // 客户门户地址
}

// SaveVerdictRequest 是保存单条验收裁定结果的请求体。
type SaveVerdictRequest struct {
	Comment string `json:"comment" validate:"max=1000"`
	Email   string `json:"email" validate:"required,email"`
	Verdict string `json:"verdict" validate:"required,oneof=fail pass"`
}

// SaveVerdictResponse 是验收裁定响应。
type SaveVerdictResponse struct {
	Verdict views.AcceptanceVerdictView `json:"verdict"`
}

// SubmitDecisionRequest 是客户提交最终验收结论的请求体。
type SubmitDecisionRequest struct {
	Comment    string           `json:"comment" validate:"max=2000"`
	Decision   string           `json:"decision" validate:"required,oneof=approve reject"`
	Email      string           `json:"email" validate:"required,email"`
	Objections []ObjectionInput `json:"objections"`
}

// ObjectionInput 是客户提出的一条异议。
type ObjectionInput struct {
	Comment      string `json:"comment" validate:"required,max=2000"`
	CriterionKey string `json:"criterionKey" validate:"required"`
}

// SubmitDecisionResponse 是最终验收结论响应。
type SubmitDecisionResponse struct {
	Delivery views.DeliveryView `json:"delivery"`
}

// PortalDeliveryResponse 是客户门户看到的交付内容。
type PortalDeliveryResponse struct {
	Delivery views.DeliveryView            `json:"delivery"`
	Evidence []views.EvidenceView          `json:"evidence"`
	Manifest views.ManifestView            `json:"manifest"`
	Verdicts []views.AcceptanceVerdictView `json:"verdicts"`
}

// ReplyObjectionRequest 是开发者回复异议的请求体。
type ReplyObjectionRequest struct {
	Content string `json:"content" validate:"required,max=2000"`
}

// ReplyObjectionResponse 是异议回复响应。
type ReplyObjectionResponse struct {
	Reply views.ObjectionReplyView `json:"reply"`
}

// ResolveObjectionResponse 是异议关闭响应。
type ResolveObjectionResponse struct {
	Objection views.ObjectionView `json:"objection"`
}

// AcceptanceSummaryResponse 是开发者视角的验收概览。
type AcceptanceSummaryResponse struct {
	Delivery   views.DeliveryView            `json:"delivery"`
	Objections []views.ObjectionView         `json:"objections"`
	Verdicts   []views.AcceptanceVerdictView `json:"verdicts"`
}
