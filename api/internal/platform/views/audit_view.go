// Package views 把数据库实体转换为对外安全的响应结构，避免暴露密码哈希与 pgtype 细节。
package views

import (
	"github.com/google/uuid"

	"api/internal/platform/db"
	"api/internal/platform/dbutil"
)

// AuditEventView 是审计事件的对外视图，只暴露时间线需要的事实字段。
//
// 变更前后的原始 JSON 不对外返回：其中可能包含内部标识与业务字段，
// 时间线只需要说明「谁在什么时候做了什么」，细节可在后续审计详情接口中按权限展开。
type AuditEventView struct {
	Action     string    `json:"action"`     // 动作标识，如 delivery.submit
	ActorID    string    `json:"actorId"`    // 操作人 ID，系统操作时为空
	ActorName  string    `json:"actorName"`  // 操作人显示名称
	ID         uuid.UUID `json:"id"`         // 审计事件 ID
	IP         string    `json:"ip"`         // 来源 IP
	ObjectID   uuid.UUID `json:"objectId"`   // 被操作对象 ID
	ObjectType string    `json:"objectType"` // 被操作对象类型
	OccurredAt string    `json:"occurredAt"` // 发生时间，上海时区
}

// FromAuditEvent 把审计实体转换为视图，并填充已解析的操作人名称。
func FromAuditEvent(model db.AuditEvent, actorName string) AuditEventView {
	return AuditEventView{
		Action:     model.Action,
		ActorID:    pgUUIDString(model.ActorID),
		ActorName:  actorName,
		ID:         model.ID,
		IP:         dbutil.StringFromText(model.Ip),
		ObjectID:   model.ObjectID,
		ObjectType: model.ObjectType,
		OccurredAt: dbutil.TimeString(model.OccurredAt),
	}
}
