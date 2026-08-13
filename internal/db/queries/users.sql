-- name: CreateUser :one
INSERT INTO users (username, password_hash, api_key_hash)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetUserByAPIKeyHash :one
SELECT * FROM users
WHERE api_key_hash = $1;

-- name: UpdateUserAPIKeyHash :one
UPDATE users
SET api_key_hash = $2
WHERE id = $1
RETURNING *;
-- name: GetUserByUsername :one
SELECT * FROM users
WHERE username = $1;
