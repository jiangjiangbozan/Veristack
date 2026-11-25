-- name: CreateEvidenceItem :one
-- 记录一条证据，文件证据保存对象键与内容摘要。
INSERT INTO evidence_items (project_id, delivery_id, requirement_id, type, title, file_key, sha256, source, note, occurred_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: GetEvidenceItem :one
-- 读取单条证据。
SELECT * FROM evidence_items
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListEvidenceByDelivery :many
-- 列出交付版本的全部证据，按发生时间倒序排列。
SELECT * FROM evidence_items
WHERE delivery_id = $1 AND deleted_at IS NULL
ORDER BY occurred_at DESC, created_at DESC;

-- name: ListEvidenceByProject :many
-- 列出项目全部证据，按发生时间倒序排列。
SELECT * FROM evidence_items
WHERE project_id = $1 AND deleted_at IS NULL
ORDER BY occurred_at DESC, created_at DESC;

-- name: CountEvidenceByDelivery :one
-- 统计交付版本的证据数量。
SELECT count(*) FROM evidence_items
WHERE delivery_id = $1 AND deleted_at IS NULL;

-- name: SoftDeleteEvidenceItem :one
-- 软删除证据记录。
UPDATE evidence_items
SET deleted_at = now()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;
-- name: CountEvidenceByProject :one
-- 统计项目证据总数。
SELECT count(*) FROM evidence_items
WHERE project_id = $1 AND deleted_at IS NULL;

