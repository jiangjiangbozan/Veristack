-- name: CreateUser :one
-- 创建用户，状态使用数据库默认值 active，返回完整实体供 Handler 构造响应。
INSERT INTO users (organization_id, email, password_hash, display_name, role)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetUserByEmail :one
-- 按邮箱读取未删除用户，供登录认证使用。
SELECT * FROM users
WHERE email = $1 AND deleted_at IS NULL
LIMIT 1;

-- name: GetUserByID :one
-- 按 ID 读取未删除用户，供权限与归属校验使用。
SELECT * FROM users
WHERE id = $1 AND deleted_at IS NULL
LIMIT 1;

-- name: ListUsersByOrganization :many
-- 分页列出组织下未删除用户，按姓名升序展示。
SELECT * FROM users
WHERE organization_id = $1 AND deleted_at IS NULL
ORDER BY display_name ASC
LIMIT $2 OFFSET $3;

-- name: CountUsersByOrganization :one
-- 统计组织下未删除用户总数，供分页元信息使用。
SELECT count(*) FROM users
WHERE organization_id = $1 AND deleted_at IS NULL;

-- name: SoftDeleteUser :exec
-- 软删除用户，保留审计事件中的操作人引用。
UPDATE users
SET deleted_at = now(), updated_at = now()
WHERE id = $1 AND deleted_at IS NULL;
