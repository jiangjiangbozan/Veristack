// Package idparse 把 URL 路径参数解析为 UUID，统一处理非法 ID 的 400 响应。
package idparse

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// FromURL 从 Chi 路由参数读取 UUID。
func FromURL(r *http.Request, name string) (uuid.UUID, error) {
	raw := chi.URLParam(r, name)
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, fmt.Errorf("路径参数 %s 不是合法 UUID", name)
	}
	return id, nil
}

// FromUUID 把可空 UUID 转为标准 UUID，空值返回错误。
func FromUUID(value pgtype.UUID) (uuid.UUID, error) {
	if !value.Valid {
		return uuid.Nil, fmt.Errorf("可空 UUID 没有有效值")
	}
	return uuid.UUID(value.Bytes), nil
}
