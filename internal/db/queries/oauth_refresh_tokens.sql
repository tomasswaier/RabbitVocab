-- name: CreateRefreshToken :one
INSERT INTO oauth_refresh_tokens (token_hash, api_key_id, user_id, client_id, expires_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetRefreshToken :one
SELECT * FROM oauth_refresh_tokens
WHERE token_hash = $1 AND expires_at > now();

-- name: DeleteRefreshToken :execrows
DELETE FROM oauth_refresh_tokens
WHERE token_hash = $1;
