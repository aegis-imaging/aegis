-- +goose Up
ALTER TABLE upload_sessions
    ADD COLUMN institution_id UUID REFERENCES institutions(id) ON DELETE SET NULL;

CREATE INDEX idx_upload_sessions_institution ON upload_sessions (institution_id);

-- +goose Down
DROP INDEX IF EXISTS idx_upload_sessions_institution;
ALTER TABLE upload_sessions DROP COLUMN IF EXISTS institution_id;
