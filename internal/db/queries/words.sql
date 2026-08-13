-- name: InsertWord :one
INSERT INTO words (language_id, native_word, learning_word, article)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetRandomWords :many
SELECT * FROM words
WHERE language_id = $1
  AND state != 'mastered'
ORDER BY
    (CASE state
        WHEN 'new'       THEN 4
        WHEN 'learning'  THEN 3
        WHEN 'confident' THEN 2
        ELSE 1
    END) * random() DESC
LIMIT $2;
-- name: UpdateWordState :one
UPDATE words
SET state = $2, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: SearchWords :many
SELECT * FROM words
WHERE language_id = $1
  AND learning_word ILIKE '%' || sqlc.arg(query)::text || '%'
ORDER BY learning_word
LIMIT 20;
-- name: DeleteWord :execrows
DELETE FROM words
WHERE words.id = $1
  AND language_id IN (SELECT id FROM languages WHERE user_id = $2);
