-- name: InsertWord :one
INSERT INTO words (language_id, native_word, learning_word, article)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetRandomWords :many
SELECT * FROM words
WHERE language_id = $1
ORDER BY random()
LIMIT $2;

-- name: UpdateWordState :one
UPDATE words
SET state = $2, updated_at = now()
WHERE id = $1
RETURNING *;
