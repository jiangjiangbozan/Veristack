// Package identity 的登录限流器，按来源与邮箱组合限制暴力尝试。
package identity

import (
	"sync"
	"time"
)

const (
	loginFailureLimit  = 5
	loginFailureWindow = 15 * time.Minute
)

type loginAttempt struct {
	first  time.Time // 窗口起点
	locked int       // 窗口内失败次数
}

// LoginLimiter 是进程内固定窗口限流器，服务重启后计数清零。
type LoginLimiter struct {
	attempts map[string]loginAttempt // 按来源 + 邮箱聚合
	mutex    sync.Mutex              // 保护 attempts
}

// NewLoginLimiter 创建登录限流器。
func NewLoginLimiter() *LoginLimiter {
	return &LoginLimiter{attempts: make(map[string]loginAttempt)}
}

// Allow 判断当前键是否仍可尝试登录，窗口过期后自动重置。
func (l *LoginLimiter) Allow(key string) bool {
	if l == nil {
		return true
	}
	l.mutex.Lock()
	defer l.mutex.Unlock()
	attempt, exists := l.attempts[key]
	if !exists || time.Since(attempt.first) >= loginFailureWindow {
		delete(l.attempts, key)
		return true
	}
	return attempt.locked < loginFailureLimit
}

// Fail 记录一次失败尝试。
func (l *LoginLimiter) Fail(key string) {
	if l == nil {
		return
	}
	l.mutex.Lock()
	defer l.mutex.Unlock()
	attempt, exists := l.attempts[key]
	if !exists || time.Since(attempt.first) >= loginFailureWindow {
		l.attempts[key] = loginAttempt{first: time.Now(), locked: 1}
		return
	}
	attempt.locked++
	l.attempts[key] = attempt
}

// Reset 在登录成功后清除计数。
func (l *LoginLimiter) Reset(key string) {
	if l == nil {
		return
	}
	l.mutex.Lock()
	defer l.mutex.Unlock()
	delete(l.attempts, key)
}
