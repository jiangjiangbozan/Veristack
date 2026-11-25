-- name: CreateProjectMember :one
-- 添加项目成员，唯一索引防止重复添加同一用户。
INSERT INTO project_members (project_id, user_id, role)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListProjectMembers :many
-- 列出项目成员，按加入时间升序展示。
SELECT pm.*, u.display_name, u.email
FROM project_members pm
JOIN users u ON u.id = pm.user_id AND u.deleted_at IS NULL
WHERE pm.project_id = $1
ORDER BY pm.created_at ASC;

-- name: DeleteProjectMember :exec
-- 移除项目成员，成员历史不保留。
DELETE FROM project_members WHERE project_id = $1 AND user_id = $2;

-- name: IsProjectMember :one
-- 判断用户是否为项目成员，供 Handler 做资源归属校验。
SELECT EXISTS(
    SELECT 1 FROM project_members WHERE project_id = $1 AND user_id = $2
) AS allowed;
