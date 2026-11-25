// Package ai 定义模型适配接口，让业务层只依赖抽象而不绑定具体厂商。
package ai

import "context"

// SuggestionKind 表示建议类型，决定了建议被采纳后作用于哪个环节。
type SuggestionKind string

const (
	// SuggestionKindRequirements 是从需求文档提取的候选需求。
	SuggestionKindRequirements SuggestionKind = "requirement_extraction"
	// SuggestionKindChangeImpact 是对需求变更的影响分析。
	SuggestionKindChangeImpact SuggestionKind = "change_impact"
	// SuggestionKindDeliverySummary 是交付版本的自然语言摘要。
	SuggestionKindDeliverySummary SuggestionKind = "delivery_summary"
)

// Requirement 是模型提出的候选需求。
type Requirement struct {
	AcceptanceCriteria []string `json:"acceptanceCriteria"`
	Code               string   `json:"code"`
	Description        string   `json:"description"`
	Priority           string   `json:"priority"`
	Title              string   `json:"title"`
}

// Result 是模型产出的建议载荷。
type Result struct {
	ImpactNotes  []string      `json:"impactNotes"`
	Requirements []Requirement `json:"requirements"`
	Summary      string        `json:"summary"`
}

// Model 是模型适配接口，实现方可以是任意大模型服务。
type Model interface {
	// AnalyzeChangeImpact 分析变更对现有需求的影响，返回影响说明。
	AnalyzeChangeImpact(ctx context.Context, baselineText, changeText string) (Result, error)
	// ExtractRequirements 从需求文档中提取候选需求，返回结构化候选。
	ExtractRequirements(ctx context.Context, document string) (Result, error)
	// Name 返回模型标识，用于记录建议来源。
	Name() string
	// SummarizeDelivery 为交付版本生成自然语言摘要。
	SummarizeDelivery(ctx context.Context, deliveryText string) (Result, error)
}
