// Package milestone 的 DTO 文件，定义里程碑接口的请求和响应结构。
package milestone

import (
	"github.com/google/uuid"

	"api/internal/platform/views"
)

// CreateMilestoneRequest 是创建里程碑的请求体。
type CreateMilestoneRequest struct {
	Name           string      `json:"name" validate:"required,max=120"`
	RequirementIDs []uuid.UUID `json:"requirementIds"`
	TargetDate     string      `json:"targetDate"`
}

// CreateMilestoneResponse 是创建里程碑响应。
type CreateMilestoneResponse struct {
	Milestone views.MilestoneView `json:"milestone"`
}

// GetMilestoneResponse 是里程碑详情响应。
type GetMilestoneResponse struct {
	Milestone views.MilestoneView `json:"milestone"`
}

// ListMilestonesResponse 是里程碑列表响应。
type ListMilestonesResponse struct {
	Items []views.MilestoneView `json:"items"`
}

// UpdateMilestoneRequest 是更新里程碑的请求体。
type UpdateMilestoneRequest struct {
	Name           string      `json:"name" validate:"required,max=120"`
	RequirementIDs []uuid.UUID `json:"requirementIds"`
	Status         string      `json:"status" validate:"required,oneof=completed pending in_progress"`
	TargetDate     string      `json:"targetDate"`
}
