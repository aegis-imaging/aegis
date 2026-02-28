-- +goose Up
ALTER TABLE studies ADD COLUMN study_date TEXT;
CREATE INDEX studies_study_date_idx ON studies (study_date) WHERE study_date IS NOT NULL;

ALTER TABLE upload_sessions ADD COLUMN study_date TEXT;

-- +goose Down
DROP INDEX IF EXISTS studies_study_date_idx;
ALTER TABLE studies DROP COLUMN IF EXISTS study_date;
ALTER TABLE upload_sessions DROP COLUMN IF EXISTS study_date;
