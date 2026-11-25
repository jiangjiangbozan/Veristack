-- name: CreateAuditEvent :exec
-- 写入审计事件，只追加不更新，保证写操作可追溯。
INSERT INTO audit_events (actor_id, action, object_type, object_id, before, after, ip)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: ListAuditEventsByObject :many
-- 列出某个对象的审计事件，按发生时间倒序展示。
SELECT * FROM audit_events
WHERE object_type = $1 AND object_id = $2
ORDER BY occurred_at DESC;
