// Package testreport 解析标准测试报告，把外部测试结果转成可汇总的结构。
package testreport

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"
)

// Case 是单个测试用例的结果。
type Case struct {
	ClassName string        `json:"className"` // 用例所属类
	Duration  time.Duration `json:"duration"`  // 耗时
	Name      string        `json:"name"`      // 用例名称
	Status    string        `json:"status"`    // 结果状态
	Suite     string        `json:"suite"`     // 所属测试套件
}

// Summary 是测试结果汇总。
type Summary struct {
	Cases   []Case `json:"cases"`   // 全部用例
	Failed  int    `json:"failed"`  // 失败数
	Passed  int    `json:"passed"`  // 通过数
	Skipped int    `json:"skipped"` // 跳过数
	Total   int    `json:"total"`   // 总数
}

// junitSuite 对应 JUnit XML 的 testsuite 节点。
type junitSuite struct {
	Tests    int         `xml:"tests,attr"`
	Failures int         `xml:"failures,attr"`
	Skipped  int         `xml:"skipped,attr"`
	Name     string      `xml:"name,attr"`
	Time     string      `xml:"time,attr"`
	Cases    []junitCase `xml:"testcase"`
}

// junitSuiteList 对应 JUnit XML 的 testsuites 根节点。
type junitSuiteList struct {
	Suites []junitSuite `xml:"testsuite"`
}

// junitCase 对应 JUnit XML 的 testcase 节点。
type junitCase struct {
	ClassName string    `xml:"classname,attr"`
	Time      string    `xml:"time,attr"`
	Name      string    `xml:"name,attr"`
	Failure   *struct{} `xml:"failure"`
	Error     *struct{} `xml:"error"`
	Skipped   *struct{} `xml:"skipped"`
}

// ParseJUnit 解析 JUnit XML 报告，兼容 testsuites 与 testsuite 两种根节点。
func ParseJUnit(content []byte) (Summary, error) {
	trimmed := strings.TrimSpace(string(content))
	if trimmed == "" {
		return Summary{}, fmt.Errorf("测试报告内容为空")
	}
	if strings.Contains(trimmed, "<testsuites") {
		return parseSuiteList(trimmed)
	}
	return parseSingleSuite(trimmed)
}

// parseSuiteList 解析以 testsuites 为根节点的报告。
func parseSuiteList(content string) (Summary, error) {
	var root junitSuiteList
	if err := xml.Unmarshal([]byte(content), &root); err != nil {
		return Summary{}, fmt.Errorf("解析测试报告失败: %w", err)
	}
	summary := Summary{Cases: []Case{}}
	for _, suite := range root.Suites {
		summary.Cases = append(summary.Cases, convertCases(suite)...)
	}
	finalizeSummary(&summary)
	return summary, nil
}

// parseSingleSuite 解析以 testsuite 为根节点的报告。
func parseSingleSuite(content string) (Summary, error) {
	var suite junitSuite
	if err := xml.Unmarshal([]byte(content), &suite); err != nil {
		return Summary{}, fmt.Errorf("解析测试报告失败: %w", err)
	}
	summary := Summary{Cases: convertCases(suite)}
	finalizeSummary(&summary)
	return summary, nil
}

// convertCases 把测试套件内的用例转换为统一结构。
func convertCases(suite junitSuite) []Case {
	cases := make([]Case, 0, len(suite.Cases))
	for _, item := range suite.Cases {
		cases = append(cases, Case{
			ClassName: item.ClassName,
			Duration:  parseDuration(item.Time),
			Name:      item.Name,
			Status:    caseStatus(item),
			Suite:     suite.Name,
		})
	}
	return cases
}

// caseStatus 依据失败、错误与跳过标记判定用例结果。
func caseStatus(item junitCase) string {
	switch {
	case item.Failure != nil || item.Error != nil:
		return "failed"
	case item.Skipped != nil:
		return "skipped"
	default:
		return "passed"
	}
}

// finalizeSummary 依据用例结果统计汇总信息。
func finalizeSummary(summary *Summary) {
	summary.Total = len(summary.Cases)
	for _, item := range summary.Cases {
		switch item.Status {
		case "failed":
			summary.Failed++
		case "skipped":
			summary.Skipped++
		default:
			summary.Passed++
		}
	}
}

// parseDuration 解析 JUnit 的秒数字符串，非法值按零处理。
func parseDuration(raw string) time.Duration {
	value, err := time.ParseDuration(raw + "s")
	if err != nil {
		return 0
	}
	return value
}

// TotalDuration 汇总全部用例耗时。
func (s Summary) TotalDuration() time.Duration {
	var total time.Duration
	for _, item := range s.Cases {
		total += item.Duration
	}
	return total
}
