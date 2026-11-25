// Package logger 基于 slog 提供结构化日志，统一时间与请求上下文字段。
package logger

import (
	"log/slog"
	"os"
	"strings"

	"api/internal/platform/datetime"
)

const (
	levelDebug = "debug"
	levelError = "error"
	levelInfo  = "info"
	levelWarn  = "warn"
)

// New 按配置级别创建 JSON 结构化日志器，时间统一使用上海时区。
func New(level string) *slog.Logger {
	options := &slog.HandlerOptions{
		Level:       parseLevel(level),
		ReplaceAttr: replaceTimeAttr,
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, options))
}

// parseLevel 将字符串级别映射为 slog 级别，未知值回退到 Info。
func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case levelDebug:
		return slog.LevelDebug
	case levelWarn:
		return slog.LevelWarn
	case levelError:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// replaceTimeAttr 把内置时间字段转换为上海时区时间。
func replaceTimeAttr(groups []string, attr slog.Attr) slog.Attr {
	if len(groups) == 0 && attr.Key == slog.TimeKey {
		return slog.Time(slog.TimeKey, attr.Value.Time().In(datetime.Shanghai()))
	}
	return attr
}
