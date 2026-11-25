-- name: CreateOrganization :one
-- 创建组织，返回完整实体供 Handler 构造响应。
INSERT INTO organizations (name)
VALUES ($1)
RETURNING *;

-- name: GetOrganization :one
-- 按 ID 读取未删除的组织，供资源归属校验使用。
SELECT * FROM organizations
WHERE id = $1 AND deleted_at IS NULL
LIMIT 1;

-- name: ListOrganizations :many
-- 分页列出未删除的组织，按名称升序便于用户查找。
SELECT * FROM organizations
WHERE deleted_at IS NULL
ORDER BY name ASC
LIMIT $1 OFFSET $2;

-- name: CountOrganizations :one
-- 统计未删除的组织总数，供分页元信息使用。
SELECT count(*) FROM organizations
WHERE deleted_at IS NULL;

-- name: SoftDeleteOrganization :exec
-- 软删除组织，保留历史交付数据可追溯。
UPDATE organizations
SET deleted_at = now(), updated_at = now()
WHERE id = $1 AND deleted_at IS NULL;
