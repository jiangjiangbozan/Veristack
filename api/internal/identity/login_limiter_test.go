// Package identity 的登录限流器单元测试。
package identity

import (
	"net/http/httptest"
	"testing"
)

func TestLoginLimiter_BlocksAfterFiveFailures(t *testing.T) {
	limiter := NewLoginLimiter()
	key := "127.0.0.1|user@example.com"

	for index := 0; index < loginFailureLimit; index++ {
		if !limiter.Allow(key) {
			t.Fatalf("第 %d 次尝试不应被拦截", index+1)
		}
		limiter.Fail(key)
	}
	if limiter.Allow(key) {
		t.Fatal("达到失败上限后应拒绝登录")
	}
	limiter.Reset(key)
	if !limiter.Allow(key) {
		t.Fatal("登录成功后应清除失败计数")
	}
}

func TestLoginAttemptKey_IgnoresForwardedFor(t *testing.T) {
	request := httptest.NewRequest("POST", "/auth/login", nil)
	request.RemoteAddr = "10.0.0.8:43120"
	request.Header.Set("X-Forwarded-For", "203.0.113.9")

	got := loginAttemptKey(request, " User@Example.com ")
	want := "10.0.0.8|user@example.com"

	if got != want {
		t.Fatalf("限流键不应信任可伪造的转发头，实际为 %q", got)
	}
}

func TestLoginLimiter_IsolatesDifferentKeys(t *testing.T) {
	limiter := NewLoginLimiter()
	first := "127.0.0.1|first@example.com"
	second := "127.0.0.1|second@example.com"

	for index := 0; index < loginFailureLimit; index++ {
		limiter.Fail(first)
	}
	if !limiter.Allow(second) {
		t.Fatal("其他邮箱不应受同一来源的失败计数影响")
	}
}
