-- name: CreateOutboxTask :one
-- 创建异步任务，先落库再由 Worker 消费，避免消息丢失。
-- dedupe_key 非空时按事件类型幂等，重复入队不会产生第二条任务。
INSERT INTO outbox_tasks (event_type, payload, project_id, delivery_id, dedupe_key)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (event_type, dedupe_key) WHERE dedupe_key <> '' DO NOTHING
RETURNING id, event_type, payload, status, attempts, next_run_at, created_at, project_id, delivery_id, dedupe_key, last_error;

-- name: ClaimPendingOutboxTasks :many
-- 领取到期的待处理任务并标记为处理中，避免多实例重复消费。
-- 邮件任务由独立 Worker 领取，这里排除 email.send，防止两边互相抢走任务。
WITH claimed AS (
    SELECT id FROM outbox_tasks
    WHERE status = 'pending' AND next_run_at <= now() AND event_type <> 'email.send'
    ORDER BY next_run_at ASC
    LIMIT $1
    FOR UPDATE SKIP LOCKED
)
UPDATE outbox_tasks SET status = 'processing', attempts = attempts + 1
WHERE id IN (SELECT id FROM claimed)
RETURNING id, event_type, payload, status, attempts, next_run_at, created_at, project_id, delivery_id, dedupe_key, last_error;

-- name: ClaimPendingEmailOutboxTasks :many
-- 邮件专用领取，只处理 email.send，和 Webhook 任务分开扫描。
WITH claimed AS (
    SELECT id FROM outbox_tasks
    WHERE status = 'pending' AND next_run_at <= now() AND event_type = 'email.send'
    ORDER BY next_run_at ASC
    LIMIT $1
    FOR UPDATE SKIP LOCKED
)
UPDATE outbox_tasks SET status = 'processing', attempts = attempts + 1
WHERE id IN (SELECT id FROM claimed)
RETURNING id, event_type, payload, status, attempts, next_run_at, created_at, project_id, delivery_id, dedupe_key, last_error;

-- name: CompleteOutboxTask :one
-- 标记任务处理成功并清空错误信息。
UPDATE outbox_tasks
SET status = 'done', next_run_at = now(), last_error = NULL
WHERE id = $1
RETURNING id, event_type, payload, status, attempts, next_run_at, created_at, project_id, delivery_id, dedupe_key, last_error;

-- name: FailOutboxTask :one
-- 标记任务处理失败并安排重试，超过上限后不再自动重试。
UPDATE outbox_tasks
SET status = $2, next_run_at = now() + make_interval(secs => $3), last_error = $4
WHERE id = $1
RETURNING id, event_type, payload, status, attempts, next_run_at, created_at, project_id, delivery_id, dedupe_key, last_error;

-- name: CountPendingOutboxTasks :one
-- 统计待处理任务数量，用于运维观测。
SELECT count(*) FROM outbox_tasks WHERE status = 'pending';
