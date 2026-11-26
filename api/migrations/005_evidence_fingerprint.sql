-- +goose Up
-- 阶段 4 可信性修复：为证据条目增加提交指纹唯一约束，
-- 使同一 Webhook 重复投递不会产生重复的提交证据。

-- 提交指纹：来自第三方事件中的提交哈希或事件唯一标识。
ALTER TABLE evidence_items ADD COLUMN IF NOT EXISTS source_fingerprint VARCHAR(128);

-- 同一项目内，同一提交指纹只允许存在一条有效证据。
CREATE UNIQUE INDEX IF NOT EXISTS evidence_items_fingerprint_key
    ON evidence_items (project_id, source_fingerprint)
    WHERE source_fingerprint IS NOT NULL AND deleted_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS evidence_items_fingerprint_key;
ALTER TABLE evidence_items DROP COLUMN IF EXISTS source_fingerprint;
