// Package views 把数据库实体转换为对外安全的响应结构，避免暴露密码哈希与 pgtype 细节。
package views

import "github.com/google/uuid"

// AcceptanceVerdictView 是单条验收条件的裁定结果。
type AcceptanceVerdictView struct {
	Comment          string    `json:"comment"`          // 裁定说明
	CriterionKey     string    `json:"criterionKey"`     // 验收条件稳定键
	DecidedAt        string    `json:"decidedAt"`        // 裁定时间
	DecidedByID      uuid.UUID `json:"decidedById"`      // 裁定人
	DeliveryID       uuid.UUID `json:"deliveryId"`       // 所属交付
	RequirementCode  string    `json:"requirementCode"`  // 所属需求编号
	RequirementTitle string    `json:"requirementTitle"` // 所属需求标题
	Verdict          string    `json:"verdict"`          // 结论：pass 或 fail
}

// ManifestCriterionView 是交付清单中的一条验收条件。
type ManifestCriterionView struct {
	Description     string `json:"description"`     // 条件描述
	Key             string `json:"key"`             // 稳定键
	RequirementCode string `json:"requirementCode"` // 所属需求编号
}

// ManifestRequirementView 是交付清单中的一条需求。
type ManifestRequirementView struct {
	Code        string                  `json:"code"`        // 需求编号
	Criteria    []ManifestCriterionView `json:"criteria"`    // 验收条件
	Description string                  `json:"description"` // 需求描述
	Priority    string                  `json:"priority"`    // 优先级
	Title       string                  `json:"title"`       // 需求标题
}

// ManifestView 是交付清单快照的对外视图。
type ManifestView struct {
	BaselineVersion int32                     `json:"baselineVersion"` // 来源基线版本
	Criteria        []ManifestCriterionView   `json:"criteria"`        // 全部验收条件
	GitRef          string                    `json:"gitRef"`          // Git 提交或标签
	ProjectName     string                    `json:"projectName"`     // 项目名称
	Requirements    []ManifestRequirementView `json:"requirements"`    // 需求快照
	Version         int32                     `json:"version"`         // 交付版本号
}

// ObjectionReplyView 是开发者对异议的回复。
type ObjectionReplyView struct {
	AuthorID    uuid.UUID `json:"authorId"`    // 回复人
	Content     string    `json:"content"`     // 回复内容
	CreatedAt   string    `json:"createdAt"`   // 回复时间
	ID          uuid.UUID `json:"id"`          // 回复 ID
	ObjectionID uuid.UUID `json:"objectionId"` // 所属异议
}

// ObjectionView 是客户提出的异议。
type ObjectionView struct {
	Content      string               `json:"content"`      // 异议内容
	CreatedAt    string               `json:"createdAt"`    // 提出时间
	CriterionKey string               `json:"criterionKey"` // 对应的验收条件
	DeliveryID   uuid.UUID            `json:"deliveryId"`   // 所属交付
	ID           uuid.UUID            `json:"id"`           // 异议 ID
	RaisedByID   uuid.UUID            `json:"raisedById"`   // 提出人
	Replies      []ObjectionReplyView `json:"replies"`      // 回复列表
	ResolvedAt   string               `json:"resolvedAt"`   // 关闭时间
	Status       string               `json:"status"`       // 状态
}
