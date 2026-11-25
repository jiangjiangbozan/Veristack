// Package evidence 的 DTO 文件，定义证据接口的请求和响应结构。
package evidence

import "api/internal/platform/views"

// CreateEvidenceRequest 是登记人工说明型证据的请求体。
type CreateEvidenceRequest struct {
	Note          string `json:"note" validate:"max=2000"`
	RequirementID string `json:"requirementId"`
	Source        string `json:"source" validate:"max=200"`
	Title         string `json:"title" validate:"required,max=200"`
	Type          string `json:"type" validate:"required,oneof=ci_report document git_commit manual recording screenshot test_report"`
}

// CreateEvidenceResponse 是登记证据的响应。
type CreateEvidenceResponse struct {
	Evidence views.EvidenceView `json:"evidence"`
}

// ListEvidenceResponse 是证据列表响应。
type ListEvidenceResponse struct {
	Items []views.EvidenceView `json:"items"`
}

// ListProjectEvidenceResponse 是项目级证据列表响应。
type ListProjectEvidenceResponse struct {
	Items []views.EvidenceView `json:"items"`
}

// UploadRequest 是上传证据附件时的表单字段。
type UploadRequest struct {
	Note          string `json:"note" form:"note" validate:"max=2000"`
	RequirementID string `json:"requirementId" form:"requirementId"`
	Source        string `json:"source" form:"source" validate:"max=200"`
	Title         string `json:"title" form:"title" validate:"required,max=200"`
	Type          string `json:"type" form:"type" validate:"required,oneof=ci_report document git_commit manual recording screenshot test_report"`
}
