-- name: CreateAPIKey :one
INSERT INTO api_keys (user_id, key_hash, label, client_id, expires_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetUserByAPIKeyHash :one
SELECT u.* FROM api_keys ak
JOIN users u ON u.id = ak.user_id
WHERE ak.key_hash = $1
  AND (ak.expires_at IS NULL OR ak.expires_at > now());

-- name: ListAPIKeysByUser :many
SELECT * FROM api_keys
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: DeleteAPIKey :execrows
DELETE FROM api_keys
WHERE id = $1 AND user_id = $2;
