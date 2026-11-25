// Package audit 把每次写操作追加到审计事件表，保证交付链路可追溯。
package audit

import (
	"context"
	"encoding/json"
	"net"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

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

// Record 追加一条审计事件，变更前后以 JSONB 保存。
func (w *Writer) Record(ctx context.Context, actor Actor, action, objectType string, objectID uuid.UUID, before, after any) error {
	beforeJSON, err := json.Marshal(before)
	if err != nil {
		return err
	}
	afterJSON, err := json.Marshal(after)
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
