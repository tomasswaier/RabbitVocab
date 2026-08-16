-- name: CreateOAuthClient :one
INSERT INTO oauth_clients (client_id, client_name, redirect_uris)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetOAuthClient :one
SELECT * FROM oauth_clients
WHERE client_id = $1;
