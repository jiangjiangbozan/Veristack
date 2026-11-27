-- +goose Up
-- 站内通知：按接收人保存，读状态与创建时间可审计。
-- 邮件发送尽力而为，失败不回滚站内通知，避免外部通道影响业务主流程。

CREATE TABLE IF NOT EXISTS notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations (id),
    recipient_id UUID NOT NULL REFERENCES users (id),
    type TEXT NOT NULL,
    title TEXT NOT NULL,
    body TEXT NOT NULL,
    link TEXT NOT NULL DEFAULT '',
    dedupe_key TEXT NOT NULL DEFAULT '',
    read_at TIMESTAMPTZ,
    emailed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS notifications_recipient_idx
    ON notifications (recipient_id, created_at DESC);
CREATE INDEX IF NOT EXISTS notifications_unread_idx
    ON notifications (recipient_id, created_at DESC) WHERE read_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS notifications_dedupe_key
    ON notifications (recipient_id, dedupe_key) WHERE dedupe_key <> '';

-- +goose Down
DROP INDEX IF EXISTS notifications_dedupe_key;
DROP INDEX IF EXISTS notifications_unread_idx;
DROP INDEX IF EXISTS notifications_recipient_idx;
DROP TABLE IF EXISTS notifications;
