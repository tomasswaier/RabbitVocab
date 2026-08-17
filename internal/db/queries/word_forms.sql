-- name: InsertWordForm :one
INSERT INTO word_forms (word_id, subject, form, tense)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetRandomWordForms :many
SELECT
    wf.id,
    wf.word_id,
    wf.subject,
    wf.form,
    wf.tense,
    wf.created_at,
    w.native_word,
    w.learning_word,
    w.state
FROM word_forms wf
JOIN words w ON w.id = wf.word_id
WHERE w.language_id = $1
  AND w.state != 'mastered'
ORDER BY
    (CASE w.state
        WHEN 'new'       THEN 4
        WHEN 'learning'  THEN 3
        WHEN 'confident' THEN 2
        ELSE 1
    END) * random() DESC
LIMIT $2;

-- name: DeleteWordForm :execrows
DELETE FROM word_forms
WHERE word_forms.id = $1
  AND word_id IN (
    SELECT w.id FROM words w
    JOIN languages l ON l.id = w.language_id
    WHERE l.user_id = $2
  );
-- name: ListWordForms :many
SELECT wf.id, wf.word_id, wf.subject, wf.form, wf.tense, wf.created_at, w.state, w.native_word
FROM word_forms wf
JOIN words w ON w.id = wf.word_id
WHERE w.language_id = $1
ORDER BY wf.created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountWordForms :one
SELECT count(*) FROM word_forms wf
JOIN words w ON w.id = wf.word_id
WHERE w.language_id = $1;
