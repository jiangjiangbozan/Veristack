-- name: CreateChangeRequest :one
-- 创建变更申请，初始状态为待确认。
INSERT INTO change_requests (project_id, baseline_id, type, reason, impact, status, requester_id)
VALUES ($1, $2, $3, $4, $5, 'pending', $6)
RETURNING *;

-- name: GetChangeRequest :one
-- 按 ID 读取变更申请，供确认流程与归属校验。
SELECT * FROM change_requests
WHERE id = $1 AND deleted_at IS NULL
LIMIT 1;

-- name: ListChangeRequestsByProject :many
-- 分页列出项目的变更申请，按创建时间倒序。
SELECT * FROM change_requests
WHERE project_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountChangeRequestsByProject :one
-- 统计项目变更申请总数，供分页元信息使用。
SELECT count(*) FROM change_requests
WHERE project_id = $1 AND deleted_at IS NULL;

-- name: DecideChangeRequest :one
-- 记录客户对变更的决定，仅待确认状态可更新。
UPDATE change_requests
SET status = $2, decided_by_id = $3, decided_at = now(), updated_at = now()
WHERE id = $1 AND status = 'pending' AND deleted_at IS NULL
RETURNING *;

-- name: SetChangeRequestSnapshot :one
-- 保存变更拟议的需求快照，确认后据此生成新的基线版本。
UPDATE change_requests
SET requirements_json = $2, updated_at = now()
WHERE id = $1 AND status = 'pending' AND deleted_at IS NULL
RETURNING *;
