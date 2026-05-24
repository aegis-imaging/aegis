-- +goose Up
-- Surface the anonymized DICOM PatientID (tag 0010,0020) so the admin
-- dashboard and DICOMweb consumers can show a human-readable identifier
-- (e.g. SUBJ-a1b2c3d4) instead of the long StudyInstanceUID.
--
-- Populated at ingest by extracting the tag from the first .dcm of a
-- study; existing rows stay NULL until the backfill endpoint runs.
ALTER TABLE studies
    ADD COLUMN IF NOT EXISTS anon_patient_id TEXT;

CREATE INDEX IF NOT EXISTS idx_studies_anon_patient_id
    ON studies(anon_patient_id)
    WHERE anon_patient_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_studies_anon_patient_id;
ALTER TABLE studies DROP COLUMN IF EXISTS anon_patient_id;
