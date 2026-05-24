-- +goose Up
--
-- Unify the two parallel subject-identifier fields. Before this migration:
--   * studies.subject_id was a user-assigned label (NULL by default).
--   * studies.anon_patient_id was the DICOM-derived pseudonymized PatientID,
--     populated automatically at ingest by client-side de-identification.
--
-- Going forward subject_id is the canonical, editable subject identifier
-- and anon_patient_id is the immutable "what DICOM said at ingest" audit
-- field. The XNAT-style subject navigation groups by subject_id; without
-- this backfill, studies uploaded before researchers explicitly assign a
-- subject_id appear under no subject at all.
--
-- This backfill copies anon_patient_id into subject_id for every row where
-- subject_id is missing. It is idempotent (re-running is a no-op) and
-- destroys no data — the original anon_patient_id values remain intact.

UPDATE studies
SET    subject_id = anon_patient_id,
       updated_at = now()
WHERE  (subject_id IS NULL OR subject_id = '')
  AND  anon_patient_id IS NOT NULL
  AND  anon_patient_id <> '';

-- +goose Down
--
-- Reverting the backfill is destructive of any researcher-assigned subject
-- merges that happened after the up migration. The safest down path is to
-- restore the NULL state only for rows whose subject_id still exactly
-- matches anon_patient_id (i.e. rows untouched since the up migration).
UPDATE studies
SET    subject_id = NULL,
       updated_at = now()
WHERE  subject_id = anon_patient_id
  AND  anon_patient_id IS NOT NULL;
