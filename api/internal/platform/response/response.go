// Package response 统一 HTTP 成功与失败响应的结构、分页格式与写入逻辑。
package response

import (
	"encoding/json"
	"net/http"
	"strconv"

	"api/internal/platform/errs"
)

const (
	defaultPage     = 1
	defaultPageSize = 20
	maxPageSize     = 100
	paramPage       = "page"
	paramPageSize   = "pageSize"
)

// Page 描述分页元信息。
type Page struct {
	Page     int   `json:"page"`     // 当前页码，从 1 开始
	PageSize int   `json:"pageSize"` // 每页条数
	Total    int64 `json:"total"`    // 总条数
}

// Pagination 是从请求解析出的分页参数。
type Pagination struct {
	Page     int // 当前页码
	PageSize int // 每页条数
}

// Envelope 是所有接口共用的响应外壳。
type Envelope struct {
	Code    string `json:"code"`           // 业务错误码，成功时为空
	Data    any    `json:"data,omitempty"` // 响应数据
	Message string `json:"message"`        // 面向用户的提示信息
	Page    *Page  `json:"page,omitempty"` // 分页元信息
	Success bool   `json:"success"`        // 是否成功
}

// Fail 写入失败响应，HTTP 状态码由 AppError 决定。
func Fail(w http.ResponseWriter, err error) {
	appErr := errs.From(err)
	write(w, appErr.Status, Envelope{
		Code:    appErr.Code,
		Message: appErr.Text,
		Success: false,
	})
}

// OK 写入成功响应。
func OK(w http.ResponseWriter, data any) {
	write(w, http.StatusOK, Envelope{Data: data, Success: true})
}

// OKWithPage 写入带分页元信息的成功响应。
func OKWithPage(w http.ResponseWriter, data any, page Page) {
	write(w, http.StatusOK, Envelope{Data: data, Page: &page, Success: true})
}

// PageOf 根据分页参数与总数构造分页元信息。
func PageOf(p Pagination, total int64) Page {
	return Page{Page: p.Page, PageSize: p.PageSize, Total: total}
}

// ParsePagination 从查询参数解析分页，非法值回退到默认值并限制上限。
func ParsePagination(r *http.Request) Pagination {
	return Pagination{
		Page:     parsePositive(r.URL.Query().Get(paramPage), defaultPage, 0),
		PageSize: parsePositive(r.URL.Query().Get(paramPageSize), defaultPageSize, maxPageSize),
	}
}

// parsePositive 解析正整数，非法或非正数时回退到默认值，limit 为 0 表示不限制上限。
func parsePositive(raw string, fallback int, limit int) int {
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		return fallback
	}
	if limit > 0 && value > limit {
		return limit
	}
	return value
}

// write 序列化并写入 JSON 响应。
func write(w http.ResponseWriter, status int, envelope Envelope) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(envelope)
}
