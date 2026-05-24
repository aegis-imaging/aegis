-- +goose Up
-- Per-series DICOM metadata captured at ingest time so the dashboard / protocol
-- service / analytics can filter and display TR / TE / protocol / etc. without
-- re-parsing DICOM files on demand.
--
-- One row per (study_id, series_instance_uid). Numeric columns are NUMERIC so
-- they can hold the "value or NULL" pattern (DICOM tags may legitimately be
-- absent on a per-series basis), and so range filters work without type casts.
--
-- The existing `study_series` table is intentionally left alone — it holds the
-- minimal (UID, description, modality, body_part, instance_count) tuple used by
-- the ingest pipeline. This table carries the heavier acquisition-parameter
-- payload that protocol-service and analytics consumers care about.

CREATE TABLE study_series_metadata (
    id                       UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    study_id                 UUID        NOT NULL REFERENCES studies(id) ON DELETE CASCADE,
    series_instance_uid      TEXT        NOT NULL,
    series_number            INT,
    series_description       TEXT,
    protocol_name            TEXT,
    modality                 TEXT,
    body_part_examined       TEXT,

    -- MR acquisition parameters (numeric for filtering / range queries)
    repetition_time          NUMERIC,        -- TR (ms)
    echo_time                NUMERIC,        -- TE (ms)
    inversion_time           NUMERIC,        -- TI (ms)
    flip_angle               NUMERIC,        -- degrees
    slice_thickness          NUMERIC,        -- mm
    spacing_between_slices   NUMERIC,        -- mm
    pixel_bandwidth          NUMERIC,        -- Hz/px
    magnetic_field_strength  NUMERIC,        -- Tesla
    echo_train_length        INT,
    number_of_averages       NUMERIC,

    -- Geometry
    rows                     INT,
    columns                  INT,
    pixel_spacing_row        NUMERIC,
    pixel_spacing_col        NUMERIC,

    -- Sequence identification
    scanning_sequence        TEXT,
    sequence_variant         TEXT,
    mr_acquisition_type      TEXT,
    sequence_name            TEXT,

    -- Device
    manufacturer             TEXT,
    manufacturer_model_name  TEXT,
    software_versions        TEXT,
    imaging_frequency        NUMERIC,

    instance_count           INT         NOT NULL DEFAULT 0,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (study_id, series_instance_uid)
);

CREATE INDEX idx_series_meta_study
    ON study_series_metadata (study_id);

CREATE INDEX idx_series_meta_protocol
    ON study_series_metadata (protocol_name)
    WHERE protocol_name IS NOT NULL;

CREATE INDEX idx_series_meta_seqdesc
    ON study_series_metadata (series_description)
    WHERE series_description IS NOT NULL;

-- +goose Down
DROP TABLE IF EXISTS study_series_metadata;
