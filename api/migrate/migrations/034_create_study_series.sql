-- Study series metadata: tracks per-series DICOM metadata for each study.
-- Populated at ingest time (batch import, DIMSE receiver) when DICOM headers are parsed.
-- Upload portal studies start with no series rows; the classification service
-- may backfill them after reading the stored DICOM files.

CREATE TABLE study_series (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    study_id            UUID        NOT NULL REFERENCES studies(id) ON DELETE CASCADE,
    series_instance_uid TEXT        NOT NULL,
    series_description  TEXT,
    modality            TEXT,
    body_part           TEXT,
    instance_count      INT         NOT NULL DEFAULT 0,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (study_id, series_instance_uid)
);

CREATE INDEX study_series_study_id_idx ON study_series (study_id);
