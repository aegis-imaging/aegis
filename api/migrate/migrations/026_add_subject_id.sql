ALTER TABLE studies ADD COLUMN subject_id TEXT;
CREATE INDEX studies_project_subject_idx ON studies (project_id, subject_id) WHERE subject_id IS NOT NULL;
