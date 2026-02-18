-- +goose Up
-- Institutions: organizations that send or receive studies
CREATE TABLE institutions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                TEXT NOT NULL,
    slug                TEXT NOT NULL UNIQUE,
    description         TEXT NOT NULL DEFAULT '',
    institution_type    TEXT NOT NULL CHECK (institution_type IN ('sender', 'receiver', 'both')),

    -- Contact
    contact_name        TEXT NOT NULL DEFAULT '',
    contact_email       TEXT NOT NULL DEFAULT '',

    -- Network identity (used to map incoming studies to an institution)
    ip_ranges           TEXT NOT NULL DEFAULT '',  -- comma-separated CIDR blocks, e.g. "10.1.0.0/16,192.168.0.0/24"
    ae_title            TEXT NOT NULL DEFAULT '',  -- DICOM AE title for DIMSE identification

    enabled             BOOLEAN NOT NULL DEFAULT TRUE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_institutions_slug    ON institutions (slug);
CREATE INDEX idx_institutions_enabled ON institutions (enabled);

-- institution_projects: which institutions participate in which projects
CREATE TABLE institution_projects (
    institution_id  UUID NOT NULL REFERENCES institutions (id) ON DELETE CASCADE,
    project_id      UUID NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    role            TEXT NOT NULL CHECK (role IN ('sender', 'receiver', 'admin')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),

    PRIMARY KEY (institution_id, project_id)
);

CREATE INDEX idx_institution_projects_project ON institution_projects (project_id);

-- Add institution_id to upload_sessions and studies so every study can be
-- traced back to the institution that submitted it (nullable — legacy rows OK).
ALTER TABLE upload_sessions ADD COLUMN institution_id UUID REFERENCES institutions (id) ON DELETE SET NULL;
ALTER TABLE studies         ADD COLUMN institution_id UUID REFERENCES institutions (id) ON DELETE SET NULL;

CREATE INDEX idx_upload_sessions_institution ON upload_sessions (institution_id);
CREATE INDEX idx_studies_institution         ON studies (institution_id);

-- +goose Down
ALTER TABLE studies         DROP COLUMN IF EXISTS institution_id;
ALTER TABLE upload_sessions DROP COLUMN IF EXISTS institution_id;
DROP TABLE IF EXISTS institution_projects;
DROP TABLE IF EXISTS institutions;
