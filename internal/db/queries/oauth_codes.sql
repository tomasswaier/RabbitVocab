-- name: CreateAuthorizationCode :one
INSERT INTO oauth_authorization_codes (code_hash, client_id, user_id, redirect_uri, code_challenge, code_challenge_method, expires_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: ConsumeAuthorizationCode :one
DELETE FROM oauth_authorization_codes
WHERE code_hash = $1 AND expires_at > now()
RETURNING *;
