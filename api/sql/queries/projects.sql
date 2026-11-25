-- name: CreateProject :one
-- 创建项目，返回完整实体供 Handler 构造响应。
INSERT INTO projects (organization_id, name, client_name, contract_no, description, start_date, end_date)
VALUES (
    sqlc.arg('organization_id'),
    sqlc.arg('name'),
    sqlc.arg('client_name'),
    sqlc.narg('contract_no'),
    sqlc.narg('description'),
    sqlc.narg('start_date'),
    sqlc.narg('end_date')
)
RETURNING *;

-- name: GetProject :one
-- 按 ID 读取未删除项目，供资源归属校验与详情展示。
SELECT * FROM projects
WHERE id = $1 AND deleted_at IS NULL
LIMIT 1;

-- name: ListProjects :many
-- 分页列出组织下未删除项目，按更新时间倒序便于先看最近变更。
SELECT * FROM projects
WHERE organization_id = $1 AND deleted_at IS NULL
ORDER BY updated_at DESC
LIMIT $2 OFFSET $3;

-- name: CountProjects :one
-- 统计组织下未删除项目总数，供分页元信息使用。
SELECT count(*) FROM projects
WHERE organization_id = $1 AND deleted_at IS NULL;

-- name: UpdateProject :one
-- 更新项目可编辑字段，冻结字段不在此列。
UPDATE projects
SET name = sqlc.arg('name'),
    client_name = sqlc.arg('client_name'),
    contract_no = sqlc.narg('contract_no'),
    description = sqlc.narg('description'),
    start_date = sqlc.narg('start_date'),
    end_date = sqlc.narg('end_date'),
    status = sqlc.arg('status'),
    archived = sqlc.arg('archived'),
    updated_at = now()
WHERE id = sqlc.arg('id') AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteProject :exec
-- 软删除项目，保留历史交付证据可追溯。
UPDATE projects
SET deleted_at = now(), updated_at = now()
WHERE id = $1 AND deleted_at IS NULL;
