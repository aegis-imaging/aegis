-- +goose Up
CREATE TABLE IF NOT EXISTS project_acl (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_email TEXT NOT NULL,
    permission TEXT NOT NULL DEFAULT 'read',  -- read, write, admin
    granted_by TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(project_id, user_email)
);
CREATE INDEX IF NOT EXISTS idx_project_acl_project ON project_acl(project_id);
CREATE INDEX IF NOT EXISTS idx_project_acl_user ON project_acl(user_email);

-- +goose Down
DROP TABLE IF EXISTS project_acl;
