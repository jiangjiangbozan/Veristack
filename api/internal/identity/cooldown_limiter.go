// Package identity 的请求冷却限流器，限制同一键在时间窗内重复触发邮件等外部副作用。
package identity

import (
	"sync"
	"time"
)

const cooldownPruneThreshold = 1024

// CooldownLimiter 记录每个键最近一次放行时间，在冷却期内拒绝重复请求。
//
// 与登录失败限流不同，它限制的是成功触发的动作频率，例如重复发送重置邮件；
// 当前为进程内实现，服务重启后自动清零。
type CooldownLimiter struct {
	entries  map[string]time.Time // 键到最近放行时间
	interval time.Duration        // 冷却时长
	mutex    sync.Mutex           // 保护 entries
	now      func() time.Time     // 当前时间，测试时可替换
}

// NewCooldownLimiter 创建指定冷却时长的限流器。
func NewCooldownLimiter(interval time.Duration) *CooldownLimiter {
	return &CooldownLimiter{
		entries:  make(map[string]time.Time),
		interval: interval,
		now:      time.Now,
	}
}

// Allow 判断键是否已过冷却期；放行时记录当前时间。
func (l *CooldownLimiter) Allow(key string) bool {
	if l == nil {
		return true
	}
	l.mutex.Lock()
	defer l.mutex.Unlock()
	now := l.now()
	l.prune(now)
	if last, exists := l.entries[key]; exists && now.Sub(last) < l.interval {
		return false
	}
	l.entries[key] = now
	return true
}

// prune 在条目过多时清理已过冷却期的键，避免内存随邮箱或用户数量无限增长。
func (l *CooldownLimiter) prune(now time.Time) {
	if len(l.entries) < cooldownPruneThreshold {
		return
	}
	for key, last := range l.entries {
		if now.Sub(last) >= l.interval {
			delete(l.entries, key)
		}
	}
}
