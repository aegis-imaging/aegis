-- +goose Up
CREATE TABLE IF NOT EXISTS auto_share_rules (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id     UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    recipient_email TEXT NOT NULL DEFAULT '',
    expiry_hours   INT NOT NULL DEFAULT 168,
    note           TEXT NOT NULL DEFAULT '',
    enabled        BOOLEAN NOT NULL DEFAULT true,
    created_by     TEXT NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_auto_share_rules_project ON auto_share_rules(project_id);

-- +goose Down
DROP TABLE IF EXISTS auto_share_rules;
