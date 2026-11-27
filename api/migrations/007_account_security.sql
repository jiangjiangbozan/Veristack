-- +goose Up
-- 账户闭环：一次性密码重置令牌。
--
-- 只保存令牌摘要，不保存明文；令牌使用一次即作废，
-- 并随密码更新把用户资料时间戳推进，使此前签发的会话令牌失效。

CREATE TABLE IF NOT EXISTS password_reset_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users (id),
    token_hash VARCHAR(64) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS password_reset_tokens_hash_key
    ON password_reset_tokens (token_hash);
CREATE INDEX IF NOT EXISTS password_reset_tokens_user_idx
    ON password_reset_tokens (user_id, created_at DESC);

-- +goose Down
DROP INDEX IF EXISTS password_reset_tokens_user_idx;
DROP INDEX IF EXISTS password_reset_tokens_hash_key;
DROP TABLE IF EXISTS password_reset_tokens;
