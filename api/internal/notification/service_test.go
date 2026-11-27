// Package notification 的纯函数单元测试，不依赖数据库与网络。
package notification

import (
	"strings"
	"testing"
)

func TestBuildEmailBody_IncludesGreetingAndLink(t *testing.T) {
	body := buildEmailBody("姜博瓒", "交付已通过验收。", "https://example.com/deliveries/1")

	if !strings.Contains(body, "你好，姜博瓒") {
		t.Fatalf("正文应包含称呼，实际为 %q", body)
	}
	if !strings.Contains(body, "交付已通过验收。") {
		t.Fatalf("正文应包含通知内容，实际为 %q", body)
	}
	if !strings.Contains(body, "查看详情：https://example.com/deliveries/1") {
		t.Fatalf("正文应包含完整链接，实际为 %q", body)
	}
}

func TestAbsoluteLink_PrefixesRelativePath(t *testing.T) {
	service := Service{PublicBaseURL: "https://veristack.example.com/"}

	got := service.absoluteLink("/deliveries/abc")

	if got != "https://veristack.example.com/deliveries/abc" {
		t.Fatalf("应补全公网地址，实际为 %q", got)
	}
}

func TestAbsoluteLink_KeepsAbsoluteAndEmptyLink(t *testing.T) {
	service := Service{PublicBaseURL: "https://veristack.example.com"}

	if got := service.absoluteLink("https://other.example.com/x"); got != "https://other.example.com/x" {
		t.Fatalf("绝对链接不应被改写，实际为 %q", got)
	}
	if got := service.absoluteLink(""); got != "" {
		t.Fatalf("空链接应保持为空，实际为 %q", got)
	}
}
