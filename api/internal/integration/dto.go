// Package integration 的 DTO 文件，定义 Webhook、CI 报告与 AI 建议接口的请求和响应结构。
package integration

// WebhookView 是 Webhook 事件的对外视图。
type WebhookView struct {
	EventType  string `json:"eventType"`  // 事件类型
	ID         string `json:"id"`         // 事件 ID
	Provider   string `json:"provider"`   // 来源平台
	ReceivedAt string `json:"receivedAt"` // 接收时间
	Status     string `json:"status"`     // 处理状态
}

// ReceiveWebhookResponse 是 Webhook 接收响应。
type ReceiveWebhookResponse struct {
	Accepted bool        `json:"accepted"` // 是否已接收
	Webhook  WebhookView `json:"webhook"`  // 已保存的事件
}

// ListWebhooksResponse 是 Webhook 事件列表响应。
type ListWebhooksResponse struct {
	Items []WebhookView `json:"items"`
}

// ImportCIReportRequest 是导入 CI 测试报告的请求体。
type ImportCIReportRequest struct {
	BuildNo    string `json:"buildNo" validate:"max=64"`
	DeliveryID string `json:"deliveryId"`
	GitRef     string `json:"gitRef" validate:"max=128"`
	Provider   string `json:"provider" validate:"required,oneof=github gitlab generic"`
	ReportXML  string `json:"reportXml" validate:"required"`
}

// ImportCIReportResponse 是 CI 报告导入响应。
type ImportCIReportResponse struct {
	Report  CIReportView    `json:"report"`
	Summary TestSummaryView `json:"summary"`
}

// CIReportView 是 CI 报告的对外视图。
type CIReportView struct {
	BuildNo   string `json:"buildNo"`   // 构建号
	CreatedAt string `json:"createdAt"` // 导入时间
	ID        string `json:"id"`        // 报告 ID
	Provider  string `json:"provider"`  // 来源平台
}

// TestSummaryView 是测试结果汇总。
type TestSummaryView struct {
	DurationMs int64 `json:"durationMs"` // 总耗时（毫秒）
	Failed     int   `json:"failed"`     // 失败数
	Passed     int   `json:"passed"`     // 通过数
	Skipped    int   `json:"skipped"`    // 跳过数
	Total      int   `json:"total"`      // 总数
}

// ListCIReportsResponse 是 CI 报告列表响应。
type ListCIReportsResponse struct {
	Items []CIReportView `json:"items"`
}

// SuggestedRequirement 是 AI 提取的候选需求。
type SuggestedRequirement struct {
	AcceptanceCriteria []string `json:"acceptanceCriteria"` // 候选验收条件
	Code               string   `json:"code"`               // 候选编号
	Description        string   `json:"description"`        // 候选描述
	Priority           string   `json:"priority"`           // 候选优先级
	Title              string   `json:"title"`              // 候选标题
}

// SuggestionContent 是 AI 建议的内容载荷。
type SuggestionContent struct {
	ImpactNotes  []string               `json:"impactNotes"`  // 影响说明
	Requirements []SuggestedRequirement `json:"requirements"` // 候选需求
	Summary      string                 `json:"summary"`      // 自然语言摘要
}

// ExtractRequirementsRequest 是从需求文档提取候选需求的请求体。
type ExtractRequirementsRequest struct {
	Content string `json:"content" validate:"required,max=200000"`
}

// ExtractRequirementsResponse 是需求提取建议响应。
type ExtractRequirementsResponse struct {
	Suggestion AISuggestionView `json:"suggestion"`
}

// AnalyzeChangeImpactRequest 是分析变更影响的请求体。
type AnalyzeChangeImpactRequest struct {
	ChangeRequestID string `json:"changeRequestId" validate:"required"`
}

// AnalyzeChangeImpactResponse 是变更影响分析建议响应。
type AnalyzeChangeImpactResponse struct {
	Suggestion AISuggestionView `json:"suggestion"`
}

// GenerateSummaryRequest 是生成交付摘要的请求体。
type GenerateSummaryRequest struct {
	DeliveryID string `json:"deliveryId" validate:"required"`
}

// GenerateSummaryResponse 是交付摘要建议响应。
type GenerateSummaryResponse struct {
	Suggestion AISuggestionView `json:"suggestion"`
}

// AISuggestionView 是 AI 建议的对外视图。
type AISuggestionView struct {
	Content   SuggestionContent `json:"content"`   // 建议内容
	CreatedAt string            `json:"createdAt"` // 生成时间
	ID        string            `json:"id"`        // 建议 ID
	Kind      string            `json:"kind"`      // 建议类型
	Model     string            `json:"model"`     // 产出模型
	Status    string            `json:"status"`    // 处理状态
}

// ListSuggestionsResponse 是 AI 建议列表响应。
type ListSuggestionsResponse struct {
	Items []AISuggestionView `json:"items"`
}
