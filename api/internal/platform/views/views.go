// Package views 把数据库实体转换为对外安全的响应结构，避免暴露密码哈希与 pgtype 细节。
package views

import (
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"api/internal/platform/db"
	"api/internal/platform/dbutil"
)

// BaselineView 是范围基线的对外视图。
type BaselineView struct {
	ChangeRequestID string            `json:"changeRequestId"` // 触发该版本的变更申请
	CreatedAt       string            `json:"createdAt"`       // 创建时间
	FrozenAt        string            `json:"frozenAt"`        // 冻结时间，空表示草稿
	ID              uuid.UUID         `json:"id"`              // 基线 ID
	ProjectID       uuid.UUID         `json:"projectId"`       // 所属项目
	Requirements    []RequirementView `json:"requirements"`    // 需求快照
	Version         int32             `json:"version"`         // 版本号
}

// ChangeRequestView 是变更申请的对外视图。
type ChangeRequestView struct {
	BaselineID  uuid.UUID `json:"baselineId"`  // 来源基线
	CreatedAt   string    `json:"createdAt"`   // 创建时间
	DecidedAt   string    `json:"decidedAt"`   // 决定时间
	ID          uuid.UUID `json:"id"`          // 变更 ID
	Impact      string    `json:"impact"`      // 影响说明
	ProjectID   uuid.UUID `json:"projectId"`   // 所属项目
	Reason      string    `json:"reason"`      // 变更原因
	RequesterID uuid.UUID `json:"requesterId"` // 发起人
	Status      string    `json:"status"`      // 状态
	Type        string    `json:"type"`        // 变更类型
}

// MilestoneView 是里程碑的对外视图。
type MilestoneView struct {
	CreatedAt      string      `json:"createdAt"`      // 创建时间
	ID             uuid.UUID   `json:"id"`             // 里程碑 ID
	Name           string      `json:"name"`           // 名称
	ProjectID      uuid.UUID   `json:"projectId"`      // 所属项目
	RequirementIDs []uuid.UUID `json:"requirementIds"` // 关联需求
	Status         string      `json:"status"`         // 状态
	TargetDate     string      `json:"targetDate"`     // 目标日期
	UpdatedAt      string      `json:"updatedAt"`      // 更新时间
}

// ProjectView 是项目的对外视图。
type ProjectView struct {
	Archived    bool      `json:"archived"`    // 是否归档
	ClientName  string    `json:"clientName"`  // 客户名称
	ContractNo  string    `json:"contractNo"`  // 合同编号
	CreatedAt   string    `json:"createdAt"`   // 创建时间
	Description string    `json:"description"` // 描述
	EndDate     string    `json:"endDate"`     // 结束日期
	ID          uuid.UUID `json:"id"`          // 项目 ID
	Name        string    `json:"name"`        // 项目名称
	StartDate   string    `json:"startDate"`   // 开始日期
	Status      string    `json:"status"`      // 状态
	UpdatedAt   string    `json:"updatedAt"`   // 更新时间
}

// RequirementView 是需求条目的对外视图。
type RequirementView struct {
	AcceptanceCriteria []string  `json:"acceptanceCriteria"` // 验收条件
	Code               string    `json:"code"`               // 需求编号
	Description        string    `json:"description"`        // 描述
	ID                 uuid.UUID `json:"id"`                 // 需求 ID
	Priority           string    `json:"priority"`           // 优先级
	Status             string    `json:"status"`             // 状态
	Title              string    `json:"title"`              // 标题
}

// NotificationView 是一条站内通知的对外视图。
type NotificationView struct {
	Body      string    `json:"body"`      // 通知正文
	CreatedAt string    `json:"createdAt"` // 创建时间
	ID        uuid.UUID `json:"id"`        // 通知 ID
	Link      string    `json:"link"`      // 站内跳转链接
	ReadAt    string    `json:"readAt"`    // 阅读时间，未读为空
	Title     string    `json:"title"`     // 通知标题
	Type      string    `json:"type"`      // 事件类型
}

// UserView 是用户的对外视图，不含密码哈希。
type UserView struct {
	CreatedAt       string    `json:"createdAt"`       // 创建时间
	DisplayName     string    `json:"displayName"`     // 显示名称
	Email           string    `json:"email"`           // 邮箱
	EmailVerifiedAt string    `json:"emailVerifiedAt"` // 邮箱验证时间，未验证为空
	ID              uuid.UUID `json:"id"`              // 用户 ID
	OrganizationID  uuid.UUID `json:"organizationId"`  // 所属组织
	Role            string    `json:"role"`            // 角色
	Status          string    `json:"status"`          // 状态
}

// AcceptanceCriterion 描述一条可验收条件。
type AcceptanceCriterion struct {
	Description string `json:"description"` // 条件描述
	ID          string `json:"id"`          // 条件标识
}

// FromBaseline 把基线实体转换为视图，并解析需求快照。
func FromBaseline(model db.ScopeBaseline) BaselineView {
	requirements := make([]RequirementView, 0)
	_ = dbutil.FromJSONBytes(model.RequirementsJson, &requirements)
	return BaselineView{
		ChangeRequestID: pgUUIDString(model.ChangeRequestID),
		CreatedAt:       dbutil.TimeString(model.CreatedAt),
		FrozenAt:        dbutil.NullableTimeString(model.FrozenAt),
		ID:              model.ID,
		ProjectID:       model.ProjectID,
		Requirements:    requirements,
		Version:         model.Version,
	}
}

// FromChangeRequest 把变更申请实体转换为视图。
func FromChangeRequest(model db.ChangeRequest) ChangeRequestView {
	return ChangeRequestView{
		BaselineID:  model.BaselineID,
		CreatedAt:   dbutil.TimeString(model.CreatedAt),
		DecidedAt:   dbutil.NullableTimeString(model.DecidedAt),
		ID:          model.ID,
		Impact:      dbutil.StringFromText(model.Impact),
		ProjectID:   model.ProjectID,
		Reason:      model.Reason,
		RequesterID: model.RequesterID,
		Status:      model.Status,
		Type:        model.Type,
	}
}

// FromMilestone 把里程碑实体转换为视图，并解析关联需求 ID。
func FromMilestone(model db.Milestone) MilestoneView {
	requirementIDs := make([]uuid.UUID, 0)
	_ = dbutil.FromJSONBytes(model.RequirementIds, &requirementIDs)
	return MilestoneView{
		CreatedAt:      dbutil.TimeString(model.CreatedAt),
		ID:             model.ID,
		Name:           model.Name,
		ProjectID:      model.ProjectID,
		RequirementIDs: requirementIDs,
		Status:         model.Status,
		TargetDate:     dbutil.DateToString(model.TargetDate),
		UpdatedAt:      dbutil.TimeString(model.UpdatedAt),
	}
}

// FromProject 把项目实体转换为视图。
func FromProject(model db.Project) ProjectView {
	return ProjectView{
		Archived:    model.Archived,
		ClientName:  model.ClientName,
		ContractNo:  dbutil.StringFromText(model.ContractNo),
		CreatedAt:   dbutil.TimeString(model.CreatedAt),
		Description: dbutil.StringFromText(model.Description),
		EndDate:     dbutil.DateToString(model.EndDate),
		ID:          model.ID,
		Name:        model.Name,
		StartDate:   dbutil.DateToString(model.StartDate),
		Status:      model.Status,
		UpdatedAt:   dbutil.TimeString(model.UpdatedAt),
	}
}

// FromRequirement 把需求实体转换为视图，并解析验收条件。
func FromRequirement(model db.Requirement) RequirementView {
	criteria := make([]string, 0)
	_ = dbutil.FromJSONBytes(model.AcceptanceCriteria, &criteria)
	return RequirementView{
		AcceptanceCriteria: criteria,
		Code:               model.Code,
		Description:        dbutil.StringFromText(model.Description),
		ID:                 model.ID,
		Priority:           model.Priority,
		Status:             model.Status,
		Title:              model.Title,
	}
}

// FromUser 把用户实体转换为不含敏感信息的视图。
func FromUser(model db.User) UserView {
	return UserView{
		CreatedAt:       dbutil.TimeString(model.CreatedAt),
		DisplayName:     model.DisplayName,
		Email:           model.Email,
		EmailVerifiedAt: dbutil.NullableTimeString(model.EmailVerifiedAt),
		ID:              model.ID,
		OrganizationID:  model.OrganizationID,
		Role:            model.Role,
		Status:          model.Status,
	}
}

// pgUUIDString 把可空 UUID 转为空串或标准字符串。
func pgUUIDString(value pgtype.UUID) string {
	if !value.Valid {
		return ""
	}
	return uuid.UUID(value.Bytes).String()
}
