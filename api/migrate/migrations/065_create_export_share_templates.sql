-- +goose Up
CREATE TABLE IF NOT EXISTS export_share_templates (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id   UUID REFERENCES projects(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    recipient_email TEXT NOT NULL DEFAULT '',
    note         TEXT NOT NULL DEFAULT '',
    expiry_hours INT NOT NULL DEFAULT 72,
    created_by   TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_export_share_templates_project ON export_share_templates(project_id);

-- +goose Down
DROP TABLE IF EXISTS export_share_templates;
