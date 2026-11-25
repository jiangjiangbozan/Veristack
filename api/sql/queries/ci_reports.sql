-- name: CreateCIReport :one
-- 保存导入的 CI 测试报告汇总。
INSERT INTO ci_reports (project_id, delivery_id, provider, build_no, git_ref, total_count, passed_count, failed_count, skipped_count, duration_ms, summary)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING *;

-- name: ListCIReportsByDelivery :many
-- 列出交付版本导入的 CI 报告，按创建时间倒序。
SELECT * FROM ci_reports
WHERE delivery_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: ListCIReportsByProject :many
-- 列出项目的 CI 报告，按创建时间倒序。
SELECT * FROM ci_reports
WHERE project_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountCIReportsByProject :one
-- 统计项目的 CI 报告总数。
SELECT count(*) FROM ci_reports
WHERE project_id = $1 AND deleted_at IS NULL;
