-- +goose Up
CREATE TABLE IF NOT EXISTS project_milestones (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    title      TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    reached    BOOLEAN NOT NULL DEFAULT false,
    reached_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_project_milestones_project ON project_milestones(project_id);

-- +goose Down
DROP TABLE IF EXISTS project_milestones;
