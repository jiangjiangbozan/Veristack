// Package views 把数据库实体转换为对外安全的响应结构，避免暴露密码哈希与 pgtype 细节。
package views

import "github.com/google/uuid"

// DeliveryView 是交付版本的对外视图。
type DeliveryView struct {
	BaselineVersion int32       `json:"baselineVersion"` // 来源基线版本
	BuildNo         string      `json:"buildNo"`         // 构建号
	CreatedAt       string      `json:"createdAt"`       // 创建时间
	GitRef          string      `json:"gitRef"`          // Git 提交或标签
	ID              uuid.UUID   `json:"id"`              // 交付 ID
	ManifestHash    string      `json:"manifestHash"`    // 交付清单摘要
	MilestoneIDs    []uuid.UUID `json:"milestoneIds"`    // 关联里程碑
	ProjectID       uuid.UUID   `json:"projectId"`       // 所属项目
	Status          string      `json:"status"`          // 状态
	UpdatedAt       string      `json:"updatedAt"`       // 更新时间
	Version         int32       `json:"version"`         // 交付版本号
}
