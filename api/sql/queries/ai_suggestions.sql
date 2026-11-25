-- name: CreateAISuggestion :one
-- 保存 AI 生成的建议草稿，默认不生效，需人工确认。
INSERT INTO ai_suggestions (project_id, kind, content, model, error_message)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetAISuggestion :one
-- 读取单条 AI 建议。
SELECT * FROM ai_suggestions WHERE id = $1 AND deleted_at IS NULL;

-- name: ListAISuggestionsByProject :many
-- 列出项目的 AI 建议，按创建时间倒序；类型传空字符串时返回全部类型。
SELECT * FROM ai_suggestions
WHERE project_id = $1 AND (kind = $2 OR $2 = '') AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: MarkAISuggestionApplied :one
-- 标记 AI 建议已被采纳。
UPDATE ai_suggestions
SET status = 'applied', applied_at = now()
WHERE id = $1
RETURNING *;

-- name: MarkAISuggestionDismissed :one
-- 标记 AI 建议已被忽略。
UPDATE ai_suggestions
SET status = 'dismissed'
WHERE id = $1
RETURNING *;
