// Package audit 把每次写操作追加到审计事件表，保证交付链路可追溯。
package audit

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"api/internal/platform/database"
	"api/internal/platform/db"
)

// Actor 描述一次写操作的发起者，由 Handler 从请求中提取。
type Actor struct {
	ID uuid.UUID // 用户 ID
	IP string    // 来源 IP
}

// FromRequest 从请求上下文与远端地址构造审计发起者。
func FromRequest(r *http.Request, actorID uuid.UUID) Actor {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	return Actor{ID: actorID, IP: host}
}

// Writer 负责把审计事件写入数据库。
type Writer struct {
	Queries *db.Queries
}

// WriterFor 构造绑定到当前请求事务的审计写入器。
//
// 审计写入与业务写入共享同一事务，审计失败会使业务变更一并回滚。
func WriterFor(ctx context.Context, pool *pgxpool.Pool) *Writer {
	return &Writer{Queries: database.Queries(ctx, pool)}
}

// Record 追加一条审计事件，变更前后以 JSONB 保存。
//
// 写入前会对敏感字段做脱敏，避免令牌、密码等机密随审计落库。
func (w *Writer) Record(ctx context.Context, actor Actor, action, objectType string, objectID uuid.UUID, before, after any) error {
	beforeJSON, err := marshalRedacted(before)
	if err != nil {
		return err
	}
	afterJSON, err := marshalRedacted(after)
	if err != nil {
		return err
	}
	return w.Queries.CreateAuditEvent(ctx, db.CreateAuditEventParams{
		ActorID:    uuidToPgtype(actor.ID),
		Action:     action,
		ObjectType: objectType,
		ObjectID:   objectID,
		Before:     beforeJSON,
		After:      afterJSON,
		Ip:         pgtype.Text{String: actor.IP, Valid: actor.IP != ""},
	})
}

// uuidToPgtype 把 UUID 转为可空 UUID，兼容审计表的可空操作人字段。
func uuidToPgtype(id uuid.UUID) pgtype.UUID {
	if id == uuid.Nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: id, Valid: true}
}

// marshalRedacted 序列化审计载荷，并把敏感字段替换为占位符。
func marshalRedacted(payload any) ([]byte, error) {
	if payload == nil {
		return nil, nil
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, err
	}
	return json.Marshal(redactValue(decoded))
}

// redactValue 递归遍历载荷，把敏感键的值替换为占位符。
func redactValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		result := make(map[string]any, len(typed))
		for key, item := range typed {
			if isSensitiveKey(key) {
				result[key] = redactedPlaceholder
				continue
			}
			result[key] = redactValue(item)
		}
		return result
	case []any:
		result := make([]any, 0, len(typed))
		for _, item := range typed {
			result = append(result, redactValue(item))
		}
		return result
	default:
		return value
	}
}

// redactedPlaceholder 是脱敏后替换敏感字段的占位符。
const redactedPlaceholder = "[redacted]"

// sensitiveKeyFragments 是需要脱敏的字段名片段，命中即替换整个值。
var sensitiveKeyFragments = []string{
	"accesstoken",
	"apikey",
	"authorization",
	"credential",
	"password",
	"privatekey",
	"refreshtoken",
	"secret",
	"signature",
	"token",
}

// isSensitiveKey 判断字段名是否属于敏感字段。
func isSensitiveKey(key string) bool {
	normalized := strings.ToLower(key)
	for _, fragment := range sensitiveKeyFragments {
		if strings.Contains(normalized, fragment) {
			return true
		}
	}
	return false
}
