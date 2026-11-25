-- name: CreateWebhook :one
-- 保存接收到的第三方 Webhook 原始事件，供后续异步处理与追溯。
INSERT INTO webhooks (provider, event_type, project_id, delivery_id, signature, payload)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetWebhook :one
-- 读取单条 Webhook 事件。
SELECT * FROM webhooks WHERE id = $1 AND deleted_at IS NULL;

-- name: ListWebhooksByProject :many
-- 列出项目收到的 Webhook 事件，按接收时间倒序。
SELECT * FROM webhooks
WHERE project_id = $1 AND deleted_at IS NULL
ORDER BY received_at DESC
LIMIT $2 OFFSET $3;

-- name: MarkWebhookProcessed :one
-- 标记 Webhook 事件已处理完成。
UPDATE webhooks
SET status = $2, processed_at = now(), error_message = $3
WHERE id = $1
RETURNING *;

-- name: CountWebhooksByProject :one
-- 统计项目收到的 Webhook 事件总数。
SELECT count(*) FROM webhooks
WHERE project_id = $1 AND deleted_at IS NULL;
