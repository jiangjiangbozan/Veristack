// Package change 的 DTO 文件，定义变更申请接口的请求和响应结构。
package change

import (
	"api/internal/platform/views"
)

// RequirementChange 描述变更中的单条需求调整。
type RequirementChange struct {
	AcceptanceCriteria []string `json:"acceptanceCriteria" validate:"dive,required,max=300"`
	Code               string   `json:"code" validate:"required,max=30"`
	Description        string   `json:"description" validate:"max=2000"`
	Operation          string   `json:"operation" validate:"required,oneof=add modify remove"`
	Priority           string   `json:"priority" validate:"required,oneof=high low medium"`
	Title              string   `json:"title" validate:"required,max=200"`
}

// CreateChangeRequestRequest 是发起变更申请的请求体。
type CreateChangeRequestRequest struct {
	Impact       string              `json:"impact" validate:"max=2000"`
	Reason       string              `json:"reason" validate:"required,max=1000"`
	Requirements []RequirementChange `json:"requirements" validate:"required,min=1,dive"`
	Type         string              `json:"type" validate:"required,oneof=add modify remove"`
}

// CreateChangeRequestResponse 是发起变更响应。
type CreateChangeRequestResponse struct {
	ChangeRequest views.ChangeRequestView `json:"changeRequest"`
}

// GetChangeRequestResponse 是变更详情响应。
type GetChangeRequestResponse struct {
	ChangeRequest views.ChangeRequestView `json:"changeRequest"`
}

// ListChangeRequestsResponse 是变更申请分页列表响应。
type ListChangeRequestsResponse struct {
	Items []views.ChangeRequestView `json:"items"`
}

// DecisionRequest 是客户对变更申请的决定请求。
type DecisionRequest struct {
	Decision string `json:"decision" validate:"required,oneof=approve reject"`
	Comment  string `json:"comment" validate:"max=1000"`
}

// DecisionResponse 是变更决定响应。
type DecisionResponse struct {
	Baseline      *views.BaselineView     `json:"baseline"`      // 批准时生成的新基线
	ChangeRequest views.ChangeRequestView `json:"changeRequest"` // 更新后的变更申请
}
