-- +goose Up
-- 阶段 4 可信性修复：客户验收身份验证。
--
-- 此前客户只要持有分享链接并填写任意邮箱即可提交验收结论，
-- 且命中其他组织已有邮箱时会被直接复用，存在冒用身份的风险。
-- 新增验证状态与一次性验证码，把「已验证邮箱」与「仅持有链接」区分开。

ALTER TABLE users ADD COLUMN IF NOT EXISTS email_verified_at TIMESTAMPTZ;

-- 客户邮箱验证码：仅保存验证码摘要，不保存明文。
CREATE TABLE IF NOT EXISTS email_verification_codes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(320) NOT NULL,
    code_hash VARCHAR(64) NOT NULL,
    attempts INT NOT NULL DEFAULT 0,
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS email_verification_codes_email_idx
    ON email_verification_codes (email, created_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS email_verification_codes_hash_key
    ON email_verification_codes (code_hash);

-- 记录客户验收身份：已验证邮箱与仅凭分享链接两种效力必须可区分。
CREATE TABLE IF NOT EXISTS acceptance_identities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    delivery_id UUID NOT NULL REFERENCES deliveries (id),
    email VARCHAR(320) NOT NULL,
    verified BOOLEAN NOT NULL DEFAULT false,
    user_id UUID REFERENCES users (id),
    verified_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS acceptance_identities_delivery_email_key
    ON acceptance_identities (delivery_id, email);

-- +goose Down
DROP INDEX IF EXISTS acceptance_identities_delivery_email_key;
DROP TABLE IF EXISTS acceptance_identities;
DROP INDEX IF EXISTS email_verification_codes_hash_key;
DROP INDEX IF EXISTS email_verification_codes_email_idx;
DROP TABLE IF EXISTS email_verification_codes;
ALTER TABLE users DROP COLUMN IF EXISTS email_verified_at;
