-- name: CreateMilestone :one
-- 创建里程碑，关联需求以 JSONB 数组保存。
INSERT INTO milestones (project_id, name, target_date, status, requirement_ids)
VALUES ($1, $2, $3, 'pending', $4)
RETURNING *;

-- name: GetMilestone :one
-- 按 ID 读取里程碑，供更新与归属校验。
SELECT * FROM milestones
WHERE id = $1 AND deleted_at IS NULL
LIMIT 1;

-- name: CountMilestonesByProject :one
-- 统计项目里程碑总数，供项目概览使用。
SELECT count(*) FROM milestones
WHERE project_id = $1 AND deleted_at IS NULL;

-- name: ListMilestonesByProject :many
-- 列出项目里程碑，按目标日期升序、创建时间升序排列。
SELECT * FROM milestones
WHERE project_id = $1 AND deleted_at IS NULL
ORDER BY target_date ASC NULLS LAST, created_at ASC;

-- name: UpdateMilestone :one
-- 更新里程碑名称、目标日期、状态与关联需求。
UPDATE milestones
SET name = $2, target_date = $3, status = $4, requirement_ids = $5, updated_at = now()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteMilestone :exec
-- 软删除里程碑，保留历史交付引用。
UPDATE milestones
SET deleted_at = now(), updated_at = now()
WHERE id = $1 AND deleted_at IS NULL;
