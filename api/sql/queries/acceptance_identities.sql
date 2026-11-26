-- name: CreateEmailVerificationCode :one
-- 保存一次性邮箱验证码，只保存验证码摘要，不保存明文。
INSERT INTO email_verification_codes (email, code_hash, expires_at)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetEmailVerificationCode :one
-- 按验证码摘要读取待验证的验证码记录。
SELECT * FROM email_verification_codes
WHERE code_hash = $1 AND consumed_at IS NULL AND expires_at > now();

-- name: ConsumeEmailVerificationCode :one
-- 标记验证码已被使用，避免同一验证码重复核验。
UPDATE email_verification_codes
SET consumed_at = now()
WHERE id = $1 AND consumed_at IS NULL
RETURNING *;

-- name: MarkUserEmailVerified :one
-- 标记用户邮箱已通过验证。
UPDATE users
SET email_verified_at = now()
WHERE id = $1
RETURNING *;

-- name: UpsertAcceptanceIdentity :one
-- 记录客户在某个交付上的验收身份，区分已验证与未验证。
INSERT INTO acceptance_identities (delivery_id, email, verified, user_id, verified_at)
VALUES ($1, $2, $3, $4, CASE WHEN $3 THEN now() ELSE NULL END)
ON CONFLICT (delivery_id, email) DO UPDATE
SET verified = acceptance_identities.verified OR EXCLUDED.verified,
    user_id = COALESCE(EXCLUDED.user_id, acceptance_identities.user_id),
    verified_at = COALESCE(acceptance_identities.verified_at, EXCLUDED.verified_at)
RETURNING *;

-- name: GetAcceptanceIdentity :one
-- 读取客户在某个交付上的验收身份。
SELECT * FROM acceptance_identities
WHERE delivery_id = $1 AND email = $2;
