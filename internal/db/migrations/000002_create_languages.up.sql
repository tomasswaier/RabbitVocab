CREATE TABLE languages (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT languages_user_id_name_unique UNIQUE (user_id, name)
);

CREATE INDEX idx_languages_user_id ON languages(user_id);
