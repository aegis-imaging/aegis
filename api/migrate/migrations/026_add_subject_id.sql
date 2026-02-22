-- +goose Up
ALTER TABLE studies ADD COLUMN subject_id TEXT;
CREATE INDEX studies_project_subject_idx ON studies (project_id, subject_id) WHERE subject_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS studies_project_subject_idx;
ALTER TABLE studies DROP COLUMN IF EXISTS subject_id;
