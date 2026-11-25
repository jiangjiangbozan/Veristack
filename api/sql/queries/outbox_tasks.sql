-- name: CreateOutboxTask :one
-- 创建异步任务，先落库再由 Worker 消费，避免消息丢失。
INSERT INTO outbox_tasks (event_type, payload, project_id, delivery_id)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ClaimPendingOutboxTasks :many
-- 领取到期的待处理任务并标记为处理中，避免多实例重复消费。
WITH claimed AS (
    SELECT id FROM outbox_tasks
    WHERE status = 'pending' AND next_run_at <= now()
    ORDER BY next_run_at ASC
    LIMIT $1
    FOR UPDATE SKIP LOCKED
)
UPDATE outbox_tasks SET status = 'processing', attempts = attempts + 1
WHERE id IN (SELECT id FROM claimed)
RETURNING *;

-- name: CompleteOutboxTask :one
-- 标记任务处理成功。
UPDATE outbox_tasks
SET status = 'done', next_run_at = now()
WHERE id = $1
RETURNING *;

-- name: FailOutboxTask :one
-- 标记任务处理失败并安排重试，超过上限后不再自动重试。
UPDATE outbox_tasks
SET status = $2, next_run_at = now() + make_interval(secs => $3)
WHERE id = $1
RETURNING *;

-- name: CountPendingOutboxTasks :one
-- 统计待处理任务数量，用于运维观测。
SELECT count(*) FROM outbox_tasks WHERE status = 'pending';
