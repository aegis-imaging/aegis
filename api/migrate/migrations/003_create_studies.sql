-- +goose Up
CREATE TABLE studies (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id          UUID NOT NULL REFERENCES projects(id),
    upload_session_id   UUID REFERENCES upload_sessions(id),

    study_instance_uid  TEXT NOT NULL UNIQUE,

    modality            TEXT NOT NULL DEFAULT '',
    body_part           TEXT NOT NULL DEFAULT '',
    study_description   TEXT NOT NULL DEFAULT '',
    series_count        INTEGER NOT NULL DEFAULT 0,
    instance_count      INTEGER NOT NULL DEFAULT 0,

    status              TEXT NOT NULL DEFAULT 'received'
                        CHECK (status IN ('received','defacing','clean','rejected')),
    defacing_required   BOOLEAN NOT NULL DEFAULT false,

    dicom_store         TEXT NOT NULL DEFAULT 'raw',

    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_studies_project ON studies (project_id);
CREATE INDEX idx_studies_status ON studies (status);

-- +goose Down
DROP TABLE studies;
