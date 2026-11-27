// Package integration 的后台任务纯函数单元测试。
package integration

import (
	"strings"
	"testing"
)

func TestDecodePushPayload_ValidAndInvalid(t *testing.T) {
	valid := []byte(`{"commits":[{"id":"abcdef1234567","message":"REQ-001 修复登录","author":"dev"}]}`)
	payload, err := decodePushPayload(valid)
	if err != nil {
		t.Fatalf("合法 push 载荷解析失败: %v", err)
	}
	if len(payload.Commits) != 1 || payload.Commits[0].Hash != "abcdef1234567" {
		t.Fatalf("提交解析结果不正确: %+v", payload.Commits)
	}
	if _, err := decodePushPayload([]byte(`{invalid`)); err == nil {
		t.Fatalf("非法 push 载荷应报错")
	}
}

func TestShortHash_TruncatesLongHash(t *testing.T) {
	longHash := "abcdef1234567890"
	if got := shortHash(longHash); got != "abcdef1" {
		t.Fatalf("长哈希应截断为 7 位，实际为 %q", got)
	}
	if got := shortHash("abc"); got != "abc" {
		t.Fatalf("短哈希应原样返回，实际为 %q", got)
	}
}

func TestRequirementCodePattern_ExtractsCodes(t *testing.T) {
	message := "feat: REQ-001 登录，关联 req-002 的导出"
	codes := requirementCodePattern.FindAllString(message, -1)
	if len(codes) != 2 {
		t.Fatalf("应提取两个需求编号，实际为 %v", codes)
	}
	if strings.ToLower(codes[0]) != "req-001" || strings.ToLower(codes[1]) != "req-002" {
		t.Fatalf("需求编号提取不正确: %v", codes)
	}
}
