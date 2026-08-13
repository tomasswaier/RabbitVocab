CREATE TABLE word_forms (
    id         BIGSERIAL PRIMARY KEY,
    word_id    BIGINT NOT NULL REFERENCES words(id) ON DELETE CASCADE,
    subject    TEXT NOT NULL,
    form       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_word_forms_word_id ON word_forms(word_id);
