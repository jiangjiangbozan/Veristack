-- name: CreateRequirement :one
-- 创建需求条目，验收条件以 JSONB 保存结构化数组。
INSERT INTO requirements (baseline_id, code, title, description, priority, status, acceptance_criteria)
VALUES ($1, $2, $3, $4, $5, 'pending', $6)
RETURNING *;

-- name: ListRequirementsByBaseline :many
-- 列出基线下的需求，按编号升序排列。
SELECT * FROM requirements
WHERE baseline_id = $1 AND deleted_at IS NULL
ORDER BY code ASC;

-- name: CountRequirementsByBaseline :one
-- 统计基线下的需求总数。
SELECT count(*) FROM requirements
WHERE baseline_id = $1 AND deleted_at IS NULL;
-- name: GetRequirementByCode :one
-- 按需求编号读取需求，用于把提交信息关联到具体需求。
SELECT r.* FROM requirements r
JOIN scope_baselines b ON b.id = r.baseline_id
WHERE r.code = $1 AND b.project_id = $2 AND r.deleted_at IS NULL
ORDER BY b.version DESC
LIMIT 1;
