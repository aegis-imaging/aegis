-- +goose Up
CREATE TABLE IF NOT EXISTS study_watchers (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    study_id   UUID NOT NULL REFERENCES studies(id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES admin_users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(study_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_study_watchers_study ON study_watchers(study_id);
CREATE INDEX IF NOT EXISTS idx_study_watchers_user ON study_watchers(user_id);

-- +goose Down
DROP TABLE IF EXISTS study_watchers;
