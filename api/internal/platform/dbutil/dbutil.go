// Package dbutil 在领域结构与 PostgreSQL 类型之间做显式转换，避免 pgtype 泄漏到接口层。
package dbutil

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"api/internal/platform/datetime"
)

const dateLayout = "2006-01-02"

// DateFromString 把 YYYY-MM-DD 字符串转为日期，空串返回 null 日期。
func DateFromString(raw string) (pgtype.Date, error) {
	if raw == "" {
		return pgtype.Date{}, nil
	}
	parsed, err := time.ParseInLocation(dateLayout, raw, datetime.Shanghai())
	if err != nil {
		return pgtype.Date{}, fmt.Errorf("日期格式无效，应为 YYYY-MM-DD: %w", err)
	}
	return pgtype.Date{Time: parsed, Valid: true}, nil
}

// DateToString 把可空日期转为 YYYY-MM-DD 字符串，null 返回空串。
func DateToString(value pgtype.Date) string {
	if !value.Valid {
		return ""
	}
	return value.Time.In(datetime.Shanghai()).Format(dateLayout)
}

// JSONBytes 把任意结构序列化为 JSONB 字节。
func JSONBytes(value any) ([]byte, error) {
	if value == nil {
		return []byte("null"), nil
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("序列化 JSONB 失败: %w", err)
	}
	return encoded, nil
}

// FromJSONBytes 把 JSONB 字节反序列化为目标结构。
func FromJSONBytes(raw []byte, target any) error {
	if len(raw) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return fmt.Errorf("反序列化 JSONB 失败: %w", err)
	}
	return nil
}

// TextFromString 把字符串转为可空文本，空串返回 null。
func TextFromString(raw string) pgtype.Text {
	if raw == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: raw, Valid: true}
}

// StringFromText 把可空文本转为字符串，null 返回空串。
func StringFromText(value pgtype.Text) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

// TimeString 把时间格式化为上海时区下的 ISO 字符串。
func TimeString(value time.Time) string {
	return value.In(datetime.Shanghai()).Format(time.RFC3339)
}

// NullableTimeString 把可空时间格式化为上海时区字符串，null 返回空串。
func NullableTimeString(value pgtype.Timestamptz) string {
	if !value.Valid {
		return ""
	}
	return TimeString(value.Time)
}
