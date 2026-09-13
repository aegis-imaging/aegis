-- +goose Up
CREATE TABLE IF NOT EXISTS study_comments (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    study_id   UUID NOT NULL REFERENCES studies(id) ON DELETE CASCADE,
    author     TEXT NOT NULL DEFAULT '',
    body       TEXT NOT NULL DEFAULT '',
    parent_id  UUID REFERENCES study_comments(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_study_comments_study_id ON study_comments(study_id);
CREATE INDEX IF NOT EXISTS idx_study_comments_parent_id ON study_comments(parent_id);

-- +goose Down
DROP TABLE IF EXISTS study_comments;
