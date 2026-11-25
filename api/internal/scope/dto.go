// Package scope 的 DTO 文件，定义需求范围基线接口的请求和响应结构。
package scope

import (
	"github.com/google/uuid"

	"api/internal/platform/views"
)

// RequirementInput 是创建基线时的单条需求输入。
type RequirementInput struct {
	AcceptanceCriteria []string `json:"acceptanceCriteria" validate:"dive,required,max=300"`
	Code               string   `json:"code" validate:"required,max=30"`
	Description        string   `json:"description" validate:"max=2000"`
	Priority           string   `json:"priority" validate:"required,oneof=high low medium"`
	Title              string   `json:"title" validate:"required,max=200"`
}

// CreateBaselineRequest 是创建并冻结基线的请求体。
type CreateBaselineRequest struct {
	Requirements []RequirementInput `json:"requirements" validate:"required,min=1,dive"`
}

// CreateBaselineResponse 是创建基线响应。
type CreateBaselineResponse struct {
	Baseline views.BaselineView `json:"baseline"`
}

// GetBaselineResponse 是基线详情响应。
type GetBaselineResponse struct {
	Baseline views.BaselineView `json:"baseline"`
}

// ListBaselinesResponse 是基线列表响应。
type ListBaselinesResponse struct {
	Items []views.BaselineView `json:"items"`
}

// DiffBaselineResponse 是两个基线之间的需求差异。
type DiffBaselineResponse struct {
	Added    []views.RequirementView `json:"added"`    // 新增需求
	Modified []views.RequirementView `json:"modified"` // 修改需求
	Removed  []views.RequirementView `json:"removed"`  // 删除需求
}

// BaselineReference 描述一条基线引用。
type BaselineReference struct {
	ID      uuid.UUID `json:"id"`      // 基线 ID
	Version int32     `json:"version"` // 版本号
}
