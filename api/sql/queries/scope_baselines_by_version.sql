-- name: GetScopeBaselineByVersion :one
-- 按版本号读取项目的需求基线，用于生成交付清单。
SELECT * FROM scope_baselines
WHERE project_id = $1 AND version = $2;
