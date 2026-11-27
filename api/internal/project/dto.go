// Package project 的 DTO 文件，定义项目与成员接口的请求和响应结构。
package project

import (
	"github.com/google/uuid"

	"api/internal/platform/views"
	"api/internal/scope"
)

// CreateProjectRequest 是创建项目的请求体。
type CreateProjectRequest struct {
	ClientName  string `json:"clientName" validate:"required,max=120"`
	ContractNo  string `json:"contractNo" validate:"max=60"`
	Description string `json:"description" validate:"max=2000"`
	EndDate     string `json:"endDate"`
	Name        string `json:"name" validate:"required,max=120"`
	StartDate   string `json:"startDate"`
}

// ImportProjectRequest 是导入已有项目的请求体，一次事务内创建项目、基线、需求与里程碑。
type ImportProjectRequest struct {
	Milestones   []ImportMilestone        `json:"milestones" validate:"dive"`   // 可选的里程碑列表
	Project      CreateProjectRequest     `json:"project" validate:"required"`  // 项目基本信息
	Requirements []scope.RequirementInput `json:"requirements" validate:"dive"` // 可选的初始需求列表
}

// ImportMilestone 是导入项目时的里程碑输入，用需求编号关联基线需求。
type ImportMilestone struct {
	Name             string   `json:"name" validate:"required,max=120"`        // 里程碑名称
	RequirementCodes []string `json:"requirementCodes" validate:"dive,max=30"` // 关联需求编号
	TargetDate       string   `json:"targetDate"`                              // 目标日期
}

// ImportProjectResponse 是项目导入结果。
type ImportProjectResponse struct {
	MilestoneCount   int               `json:"milestoneCount"`   // 创建的里程碑数量
	Project          views.ProjectView `json:"project"`          // 创建的项目
	RequirementCount int               `json:"requirementCount"` // 创建的需求数量
}

// CreateProjectResponse 是创建项目响应。
type CreateProjectResponse struct {
	Project views.ProjectView `json:"project"`
}

// GetProjectResponse 是项目详情响应。
type GetProjectResponse struct {
	Project views.ProjectView `json:"project"`
}

// ListProjectsResponse 是项目分页列表响应。
type ListProjectsResponse struct {
	Items []views.ProjectView `json:"items"`
}

// UpdateProjectRequest 是更新项目的请求体。
type UpdateProjectRequest struct {
	Archived    bool   `json:"archived"`
	ClientName  string `json:"clientName" validate:"required,max=120"`
	ContractNo  string `json:"contractNo" validate:"max=60"`
	Description string `json:"description" validate:"max=2000"`
	EndDate     string `json:"endDate"`
	Name        string `json:"name" validate:"required,max=120"`
	StartDate   string `json:"startDate"`
	Status      string `json:"status" validate:"required,oneof=active completed archived"`
}

// UpdateProjectResponse 是更新项目响应。
type UpdateProjectResponse struct {
	Project views.ProjectView `json:"project"`
}

// ProjectOverviewResponse 是项目概览统计响应。
type ProjectOverviewResponse struct {
	BaselineCount      int64 `json:"baselineCount"`      // 范围基线版本数
	ChangeRequestCount int64 `json:"changeRequestCount"` // 变更申请数
	MilestoneCount     int64 `json:"milestoneCount"`     // 里程碑数
	RequirementCount   int64 `json:"requirementCount"`   // 最新基线需求数
}

// AddMemberRequest 是添加项目成员的请求体。
type AddMemberRequest struct {
	Email string `json:"email" validate:"required,email"`
	Role  string `json:"role" validate:"required,oneof=developer reviewer"`
}

// AddMemberResponse 是添加成员响应。
type AddMemberResponse struct {
	Member MemberView `json:"member"`
}

// ListMembersResponse 是项目成员列表响应。
type ListMembersResponse struct {
	Items []MemberView `json:"items"`
}

// MemberView 是项目成员视图。
type MemberView struct {
	DisplayName string    `json:"displayName"` // 显示名称
	Email       string    `json:"email"`       // 邮箱
	ID          uuid.UUID `json:"id"`          // 用户 ID
	Role        string    `json:"role"`        // 项目角色
}
