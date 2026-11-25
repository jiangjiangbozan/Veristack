-- name: CreateDelivery :one
-- 创建交付版本，交付清单快照随版本一同冻结。
INSERT INTO deliveries (project_id, version, baseline_version, milestone_ids, git_ref, build_no, status, manifest_json, manifest_hash)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: GetDelivery :one
-- 读取单个交付版本。
SELECT * FROM deliveries
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListDeliveriesByProject :many
-- 列出项目交付版本，按版本号倒序排列。
SELECT * FROM deliveries
WHERE project_id = $1 AND deleted_at IS NULL
ORDER BY version DESC
LIMIT $2 OFFSET $3;

-- name: CountDeliveriesByProject :one
-- 统计项目交付版本总数。
SELECT count(*) FROM deliveries
WHERE project_id = $1 AND deleted_at IS NULL;

-- name: SubmitDelivery :one
-- 提交交付进入待验收状态，仅草稿状态可提交。
UPDATE deliveries
SET status = 'pending', updated_at = now()
WHERE id = $1 AND status = 'draft' AND deleted_at IS NULL
RETURNING *;

-- name: UpdateDeliveryStatus :one
-- 更新交付验收状态。
UPDATE deliveries
SET status = $2, updated_at = now()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteDelivery :one
-- 软删除交付版本。
UPDATE deliveries
SET deleted_at = now()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;
