package ai

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

// requirementPattern 匹配形如「需求 3：用户可以导出报表」或「3. 用户可以导出报表」的条目。
// 空白匹配限定在行内，避免跨越换行把多条需求合并成一条。
var requirementPattern = regexp.MustCompile(`(?m)^[^\S\r\n]*(?:需求[^\S\r\n]*)?(\d+)[.、)）:：][^\S\r\n]*(.+)$`)

// heuristicModel 在未配置模型密钥时提供确定性的本地建议，保证功能可用且结果可复现。
type heuristicModel struct{}

// NewHeuristicModel 创建本地确定性模型适配器。
func NewHeuristicModel() *heuristicModel {
	return &heuristicModel{}
}

// AnalyzeChangeImpact 依据变更前后文本的差异给出确定性的影响说明。
func (m *heuristicModel) AnalyzeChangeImpact(_ context.Context, baselineText, changeText string) (Result, error) {
	baselineCodes := extractCodes(baselineText)
	changeCodes := extractCodes(changeText)
	notes := make([]string, 0)
	for code := range changeCodes {
		if baselineCodes[code] {
			notes = append(notes, fmt.Sprintf("变更涉及已存在的需求 %s，建议确认其验收条件是否需要同步调整。", code))
			continue
		}
		notes = append(notes, fmt.Sprintf("变更引入了新需求 %s，需要确认是否纳入范围基线并补充验收条件。", code))
	}
	if len(notes) == 0 {
		notes = append(notes, "未在变更内容中识别到明确的需求编号，建议人工确认影响范围。")
	}
	return Result{ImpactNotes: notes, Requirements: []Requirement{}, Summary: strings.Join(notes, " ")}, nil
}

// ExtractRequirements 按条目编号抽取候选需求，并生成建议的验收条件占位。
func (m *heuristicModel) ExtractRequirements(_ context.Context, document string) (Result, error) {
	matches := requirementPattern.FindAllStringSubmatch(document, -1)
	requirements := make([]Requirement, 0, len(matches))
	for _, match := range matches {
		if len(match) < 3 {
			continue
		}
		title := strings.TrimSpace(match[2])
		if title == "" {
			continue
		}
		requirements = append(requirements, Requirement{
			AcceptanceCriteria: []string{fmt.Sprintf("%s 被正确实现并通过验收", title)},
			Code:               fmt.Sprintf("REQ-%03d", len(requirements)+1),
			Description:        title,
			Priority:           "medium",
			Title:              title,
		})
	}
	summary := fmt.Sprintf("从文档中识别到 %d 条候选需求，验收条件需人工确认后才会进入基线。", len(requirements))
	return Result{ImpactNotes: []string{}, Requirements: requirements, Summary: summary}, nil
}

// Name 返回模型标识。
func (m *heuristicModel) Name() string {
	return "local-heuristic"
}

// SummarizeDelivery 依据交付内容生成确定性的结构化摘要。
func (m *heuristicModel) SummarizeDelivery(_ context.Context, deliveryText string) (Result, error) {
	lines := make([]string, 0)
	for _, line := range strings.Split(deliveryText, "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			lines = append(lines, trimmed)
		}
	}
	summary := "本次交付包含以下内容：\n" + strings.Join(lines, "\n")
	return Result{ImpactNotes: []string{}, Requirements: []Requirement{}, Summary: summary}, nil
}

// extractCodes 提取文本中形如 REQ-001 的需求编号集合。
func extractCodes(text string) map[string]bool {
	codes := make(map[string]bool)
	for _, match := range regexp.MustCompile(`(?i)REQ-\d{3}`).FindAllString(text, -1) {
		codes[strings.ToUpper(match)] = true
	}
	return codes
}
