-- +goose Up
CREATE TABLE IF NOT EXISTS study_tags (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    study_id   UUID NOT NULL REFERENCES studies(id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    tag        TEXT NOT NULL,
    created_by TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(study_id, tag)
);
CREATE INDEX IF NOT EXISTS idx_study_tags_study ON study_tags(study_id);
CREATE INDEX IF NOT EXISTS idx_study_tags_project_tag ON study_tags(project_id, tag);

-- +goose Down
DROP TABLE IF EXISTS study_tags;
