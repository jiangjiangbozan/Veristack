-- name: CreateShareToken :one
-- 创建客户分享链接，库中只保存令牌的摘要。
INSERT INTO share_tokens (delivery_id, token_hash, expires_at)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetShareTokenByHash :one
-- 按令牌摘要查找有效的分享链接。
SELECT * FROM share_tokens
WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > now();

-- name: RevokeShareTokens :one
-- 作废交付下全部分享链接，避免旧链接继续可用。
UPDATE share_tokens
SET revoked_at = now()
WHERE delivery_id = $1 AND revoked_at IS NULL
RETURNING *;

-- name: ListShareTokensByDelivery :many
-- 列出交付版本的分享链接状态。
SELECT * FROM share_tokens
WHERE delivery_id = $1
ORDER BY created_at DESC;
