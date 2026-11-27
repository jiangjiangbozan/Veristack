-- +goose Up
-- Outbox 任务增加幂等键与最近一次失败原因，供通知/邮件异步投递重试与运维排查。
ALTER TABLE outbox_tasks
    ADD COLUMN IF NOT EXISTS dedupe_key TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS last_error TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS outbox_tasks_event_dedupe_key
    ON outbox_tasks (event_type, dedupe_key) WHERE dedupe_key <> '';

-- 邮件任务单独走一个部分索引，避免与 Webhook 任务争抢扫描窗口。
CREATE INDEX IF NOT EXISTS outbox_tasks_email_pending_idx
    ON outbox_tasks (next_run_at) WHERE status = 'pending' AND event_type = 'email.send';

-- +goose Down
DROP INDEX IF EXISTS outbox_tasks_email_pending_idx;
DROP INDEX IF EXISTS outbox_tasks_event_dedupe_key;
ALTER TABLE outbox_tasks
    DROP COLUMN IF EXISTS last_error,
    DROP COLUMN IF EXISTS dedupe_key;
