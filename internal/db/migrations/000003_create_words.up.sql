CREATE TYPE word_state AS ENUM ('new', 'learning', 'confident', 'mastered');

CREATE TABLE words (
    id            BIGSERIAL PRIMARY KEY,
    language_id   BIGINT NOT NULL REFERENCES languages(id) ON DELETE CASCADE,
    native_word   TEXT NOT NULL,
    learning_word TEXT NOT NULL,
    article       TEXT,
    state         word_state NOT NULL DEFAULT 'new',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_words_language_id_state ON words(language_id, state);
