-- +goose Up

CREATE TABLE anon_profiles (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id    UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name          TEXT NOT NULL,
    description   TEXT NOT NULL DEFAULT '',
    retained_tags JSONB NOT NULL DEFAULT '[]',
    enabled       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(project_id, name)
);

CREATE INDEX idx_anon_profiles_project ON anon_profiles(project_id);

ALTER TABLE projects
    ADD COLUMN default_anon_profile_id UUID REFERENCES anon_profiles(id) ON DELETE SET NULL;

-- +goose Down

ALTER TABLE projects DROP COLUMN IF EXISTS default_anon_profile_id;
DROP TABLE IF EXISTS anon_profiles;
