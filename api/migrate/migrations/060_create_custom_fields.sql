-- +goose Up

-- Per-project custom field definitions.
CREATE TABLE IF NOT EXISTS custom_field_definitions (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    field_type TEXT NOT NULL DEFAULT 'text',  -- text, number, date, select
    options    JSONB NOT NULL DEFAULT '[]',   -- choices for 'select' type
    required   BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(project_id, name)
);
CREATE INDEX IF NOT EXISTS idx_custom_field_defs_project ON custom_field_definitions(project_id);

-- Per-study custom field values.
CREATE TABLE IF NOT EXISTS study_custom_field_values (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    study_id   UUID NOT NULL REFERENCES studies(id) ON DELETE CASCADE,
    field_id   UUID NOT NULL REFERENCES custom_field_definitions(id) ON DELETE CASCADE,
    value      TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(study_id, field_id)
);
CREATE INDEX IF NOT EXISTS idx_custom_field_values_study ON study_custom_field_values(study_id);

-- +goose Down
DROP TABLE IF EXISTS study_custom_field_values;
DROP TABLE IF EXISTS custom_field_definitions;
