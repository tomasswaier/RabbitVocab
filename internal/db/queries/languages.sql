-- name: CreateLanguage :one
INSERT INTO languages (user_id, name)
VALUES ($1, $2)
RETURNING *;

-- name: ListLanguagesByUser :many
SELECT * FROM languages
WHERE user_id = $1
ORDER BY name;

-- name: CountLanguagesByUser :one
SELECT count(*) FROM languages
WHERE user_id = $1;
