-- name: CreateObjection :one
-- 记录客户对某条验收条件提出的异议。
INSERT INTO objections (delivery_id, criterion_id, raised_by_id, content)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListObjectionsByDelivery :many
-- 列出交付版本的全部异议，按提出时间正序排列。
SELECT * FROM objections
WHERE delivery_id = $1
ORDER BY created_at;

-- name: GetObjection :one
-- 读取单条异议。
SELECT * FROM objections
WHERE id = $1;

-- name: ResolveObjection :one
-- 关闭异议并记录关闭时间。
UPDATE objections
SET status = 'resolved', resolved_at = now()
WHERE id = $1 AND status = 'open'
RETURNING *;

-- name: CountOpenObjections :one
-- 统计交付版本中待处理的异议数量。
SELECT count(*) FROM objections
WHERE delivery_id = $1 AND status = 'open';

-- name: CreateObjectionReply :one
-- 记录开发者对异议的回复。
INSERT INTO objection_replies (objection_id, author_id, content)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListObjectionReplies :many
-- 列出异议的全部回复，按时间正序排列。
SELECT * FROM objection_replies
WHERE objection_id = $1
ORDER BY created_at;
