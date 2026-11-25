// Package delivery 的 DTO 文件，定义交付接口的请求和响应结构。
package delivery

import (
	"api/internal/platform/views"
)

// CreateDeliveryRequest 是创建交付版本的请求体。
type CreateDeliveryRequest struct {
	BaselineVersion int32    `json:"baselineVersion" validate:"required,min=1"`
	BuildNo         string   `json:"buildNo" validate:"max=60"`
	GitRef          string   `json:"gitRef" validate:"max=120"`
	MilestoneIDs    []string `json:"milestoneIds"`
	Version         int32    `json:"version" validate:"required,min=1"`
}

// CreateDeliveryResponse 是创建交付版本的响应。
type CreateDeliveryResponse struct {
	Delivery views.DeliveryView `json:"delivery"`
}

// GetDeliveryResponse 是交付详情的响应。
type GetDeliveryResponse struct {
	Delivery views.DeliveryView `json:"delivery"`
}

// ListDeliveriesResponse 是交付版本列表响应。
type ListDeliveriesResponse struct {
	Items []views.DeliveryView `json:"items"`
}

// ManifestCriterion 是交付清单中的单条验收条件。
type ManifestCriterion struct {
	Description     string `json:"description"`     // 条件描述
	Key             string `json:"key"`             // 稳定键
	RequirementCode string `json:"requirementCode"` // 所属需求编号
}

// ManifestRequirement 是交付清单中的单条需求。
type ManifestRequirement struct {
	Code        string              `json:"code"`        // 需求编号
	Criteria    []ManifestCriterion `json:"criteria"`    // 验收条件
	Description string              `json:"description"` // 需求描述
	Priority    string              `json:"priority"`    // 优先级
	Title       string              `json:"title"`       // 需求标题
}

// Manifest 是交付清单快照。
type Manifest struct {
	BaselineVersion int32                 `json:"baselineVersion"` // 来源基线版本
	Criteria        []ManifestCriterion   `json:"criteria"`        // 全部验收条件
	GitRef          string                `json:"gitRef"`          // Git 提交或标签
	ProjectName     string                `json:"projectName"`     // 项目名称
	Requirements    []ManifestRequirement `json:"requirements"`    // 需求快照
	Version         int32                 `json:"version"`         // 交付版本号
}

// SubmitDeliveryResponse 是提交验收的响应。
type SubmitDeliveryResponse struct {
	Delivery views.DeliveryView `json:"delivery"`
}

// PackResponse 是导出 Evidence Pack 的响应，给出打包文件与清单摘要。
type PackResponse struct {
	DownloadURL string `json:"downloadUrl"`    // 打包文件下载地址
	ManifestSHA string `json:"manifestSha256"` // 交付清单摘要
	PackPath    string `json:"packPath"`       // Pack 在服务端的存放路径
	Signed      bool   `json:"signed"`         // 是否附带签名
}
