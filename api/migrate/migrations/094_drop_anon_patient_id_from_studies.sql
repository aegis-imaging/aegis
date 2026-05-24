-- +goose Up
--
-- Retire the anon_patient_id audit column.
--
-- PR #499 unified subject_id and anon_patient_id at the API/UI layer:
-- subject_id became the canonical, editable subject identifier, while
-- anon_patient_id was kept as an immutable "what DICOM said at ingest"
-- audit trail.
--
-- Operational experience showed the dual-column model wasn't paying for
-- itself: the only consumer of anon_patient_id was the admin Studies
-- table's Patient column, and the per-study audit trail it provided is
-- already captured by the audit log table for any subject_id edits.
-- Carrying two columns added cognitive overhead, query surface area, and
-- one more place to forget to keep in sync.
--
-- This migration drops the column. The DROP COLUMN cascades to the
-- single-column index added in migration 091. Backfill of subject_id
-- for legacy rows runs as a separate admin tool (POST /api/admin/
-- backfill-study-metadata) since it must re-read DICOM headers from
-- object storage.

ALTER TABLE studies DROP COLUMN IF EXISTS anon_patient_id;

-- +goose Down
--
-- Rolling back recreates the column and its index, but the original
-- audit values are NOT recoverable — they lived only in this column.
-- Down is for testing the migration framework, not for production
-- recovery.

ALTER TABLE studies ADD COLUMN IF NOT EXISTS anon_patient_id TEXT;

CREATE INDEX IF NOT EXISTS idx_studies_anon_patient_id
    ON studies(anon_patient_id)
    WHERE anon_patient_id IS NOT NULL;
