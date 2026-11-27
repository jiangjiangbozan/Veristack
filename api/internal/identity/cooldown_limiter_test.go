package identity

import (
	"testing"
	"time"
)

// TestCooldownLimiter_BlocksWithinInterval 验证冷却期内重复请求被拒绝。
func TestCooldownLimiter_BlocksWithinInterval(t *testing.T) {
	now := time.Date(2026, 11, 26, 12, 0, 0, 0, time.UTC)
	limiter := NewCooldownLimiter(time.Minute)
	limiter.now = func() time.Time { return now }

	if !limiter.Allow("forgot|a@example.com") {
		t.Fatal("首次请求应放行")
	}
	if limiter.Allow("forgot|a@example.com") {
		t.Fatal("冷却期内重复请求应被拒绝")
	}
	if !limiter.Allow("forgot|b@example.com") {
		t.Fatal("不同键不应互相影响")
	}
}

// TestCooldownLimiter_AllowsAfterInterval 验证冷却期结束后重新放行。
func TestCooldownLimiter_AllowsAfterInterval(t *testing.T) {
	now := time.Date(2026, 11, 26, 12, 0, 0, 0, time.UTC)
	limiter := NewCooldownLimiter(time.Minute)
	limiter.now = func() time.Time { return now }

	if !limiter.Allow("verify|user-1") {
		t.Fatal("首次请求应放行")
	}
	now = now.Add(time.Minute)
	if !limiter.Allow("verify|user-1") {
		t.Fatal("冷却期结束后应重新放行")
	}
}

// TestCooldownLimiter_NilAllows 验证未注入限流器时不阻断业务。
func TestCooldownLimiter_NilAllows(t *testing.T) {
	var limiter *CooldownLimiter
	if !limiter.Allow("any") {
		t.Fatal("nil 限流器应默认放行")
	}
}
