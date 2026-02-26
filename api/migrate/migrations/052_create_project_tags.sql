-- +goose Up
CREATE TABLE IF NOT EXISTS project_tags (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    tag        TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(project_id, tag)
);
CREATE INDEX IF NOT EXISTS idx_project_tags_project ON project_tags(project_id);

-- +goose Down
DROP TABLE IF EXISTS project_tags;
