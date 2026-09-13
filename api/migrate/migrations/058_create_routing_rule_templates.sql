-- +goose Up
CREATE TABLE IF NOT EXISTS routing_rule_templates (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    action      TEXT NOT NULL,
    modality    TEXT,
    body_part   TEXT,
    source      TEXT,
    created_by  TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS routing_rule_templates;
