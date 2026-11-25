-- name: CreateScopeBaseline :one
-- 创建范围基线版本，版本由 Service 在事务内计算后传入。
INSERT INTO scope_baselines (project_id, version, requirements_json)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetScopeBaseline :one
-- 按 ID 读取基线，供详情与变更校验使用。
SELECT * FROM scope_baselines
WHERE id = $1
LIMIT 1;

-- name: CountScopeBaselines :one
-- 统计项目基线版本总数，供项目概览使用。
SELECT count(*) FROM scope_baselines WHERE project_id = $1;

-- name: GetLatestBaseline :one
-- 读取项目最新版本基线，作为变更与交付的默认来源。
SELECT * FROM scope_baselines
WHERE project_id = $1
ORDER BY version DESC
LIMIT 1;

-- name: ListScopeBaselines :many
-- 列出项目全部基线，按版本升序展示演进过程。
SELECT * FROM scope_baselines
WHERE project_id = $1
ORDER BY version ASC;

-- name: FreezeScopeBaseline :one
-- 冻结基线并记录触发它的变更申请，已冻结基线不可再次编辑。
UPDATE scope_baselines
SET frozen_at = now(), change_request_id = $2
WHERE id = $1 AND frozen_at IS NULL
RETURNING *;
