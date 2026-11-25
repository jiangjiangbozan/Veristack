// Package datetime 统一提供上海时区与日期边界计算，避免依赖运行环境默认时区。
package datetime

import "time"

const shanghaiOffsetSeconds = 8 * 60 * 60

// Shanghai 返回系统统一使用的上海时区，加载失败时回退到固定偏移。
func Shanghai() *time.Location {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.FixedZone("Asia/Shanghai", shanghaiOffsetSeconds)
	}
	return location
}

// Now 返回上海时区下的当前时间。
func Now() time.Time {
	return time.Now().In(Shanghai())
}

// StartOfDay 返回给定时间在上海时区下的当天零点。
func StartOfDay(t time.Time) time.Time {
	local := t.In(Shanghai())
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, Shanghai())
}
