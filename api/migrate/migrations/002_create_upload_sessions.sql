-- +goose Up
CREATE TYPE upload_status AS ENUM (
    'initiated',
    'uploaded',
    'ingesting',
    'completed',
    'failed'
);

CREATE TABLE upload_sessions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id      UUID NOT NULL REFERENCES projects(id),
    status          upload_status NOT NULL DEFAULT 'initiated',
    file_count      INTEGER NOT NULL DEFAULT 0,
    storage_prefix  TEXT NOT NULL,
    uploader_ip     TEXT NOT NULL DEFAULT '',
    uploader_email  TEXT NOT NULL DEFAULT '',

    study_instance_uid TEXT,
    modality           TEXT,
    body_part          TEXT,

    error_message   TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_upload_sessions_project ON upload_sessions (project_id);
CREATE INDEX idx_upload_sessions_status ON upload_sessions (status);

-- +goose Down
DROP TABLE upload_sessions;
DROP TYPE upload_status;
