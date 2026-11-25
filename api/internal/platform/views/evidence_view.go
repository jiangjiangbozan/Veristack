// Package views 把数据库实体转换为对外安全的响应结构，避免暴露密码哈希与 pgtype 细节。
package views

import "github.com/google/uuid"

// EvidenceView 是证据条目的对外视图。
type EvidenceView struct {
	CreatedAt     string    `json:"createdAt"`     // 记录时间
	FileKey       string    `json:"fileKey"`       // 对象存储键
	ID            uuid.UUID `json:"id"`            // 证据 ID
	Note          string    `json:"note"`          // 人工说明内容
	OccurredAt    string    `json:"occurredAt"`    // 证据发生时间
	ProjectID     uuid.UUID `json:"projectId"`     // 所属项目
	RequirementID string    `json:"requirementId"` // 关联需求
	SHA256        string    `json:"sha256"`        // 文件内容摘要
	Source        string    `json:"source"`        // 证据来源
	Title         string    `json:"title"`         // 标题
	Type          string    `json:"type"`          // 证据类型
}
