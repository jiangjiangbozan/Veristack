// Package delivery 的业务逻辑单元测试，不依赖数据库与网络。
package delivery

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"api/internal/platform/db"
)

func TestMergeAuditEvents_OrdersByTimeDesc(t *testing.T) {
	base := time.Date(2026, 11, 26, 10, 0, 0, 0, time.UTC)
	deliveryEvents := []db.AuditEvent{
		{Action: "delivery.create", OccurredAt: base},
		{Action: "delivery.export", OccurredAt: base.Add(2 * time.Hour)},
	}
	acceptanceEvents := []db.AuditEvent{
		{Action: "acceptance.share", OccurredAt: base.Add(time.Hour)},
	}

	events := mergeAuditEvents(deliveryEvents, acceptanceEvents)

	want := []string{"delivery.export", "acceptance.share", "delivery.create"}
	if len(events) != len(want) {
		t.Fatalf("事件数量应为 %d，实际为 %d", len(want), len(events))
	}
	for index, action := range want {
		if events[index].Action != action {
			t.Fatalf("第 %d 条事件应为 %s，实际为 %s", index, action, events[index].Action)
		}
	}
}

func TestMergeAuditEvents_SameTimeKeepsOrder(t *testing.T) {
	occurredAt := time.Date(2026, 11, 26, 10, 0, 0, 0, time.UTC)
	first := []db.AuditEvent{{Action: "delivery.create", OccurredAt: occurredAt}}
	second := []db.AuditEvent{{Action: "acceptance.share", OccurredAt: occurredAt}}

	events := mergeAuditEvents(first, second)

	if events[0].Action != "delivery.create" || events[1].Action != "acceptance.share" {
		t.Fatalf("同一时刻应保持分组顺序，实际为 %v", []string{events[0].Action, events[1].Action})
	}
}

func TestAuditActorKey_ReturnsStringOrEmpty(t *testing.T) {
	actorID := uuid.New()
	valid := pgtype.UUID{Bytes: [16]byte(actorID), Valid: true}

	if key := auditActorKey(valid); key != actorID.String() {
		t.Fatalf("有效操作人应返回 UUID 字符串，实际为 %s", key)
	}
	if key := auditActorKey(pgtype.UUID{}); key != "" {
		t.Fatalf("系统操作应返回空串，实际为 %s", key)
	}
}
