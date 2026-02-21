-- +goose Up

-- Add 'approved' to the status state machine.
-- PostgreSQL requires dropping and re-adding the inline CHECK constraint.
ALTER TABLE studies DROP CONSTRAINT IF EXISTS studies_status_check;
ALTER TABLE studies ADD CONSTRAINT studies_status_check
    CHECK (status IN ('received', 'defacing', 'clean', 'approved', 'rejected'));

-- Track whether a study originated from an external upload or internal enterprise ingest.
ALTER TABLE studies ADD COLUMN IF NOT EXISTS source TEXT NOT NULL DEFAULT 'external'
    CHECK (source IN ('external', 'internal'));

-- +goose Down
ALTER TABLE studies DROP CONSTRAINT IF EXISTS studies_status_check;
ALTER TABLE studies ADD CONSTRAINT studies_status_check
    CHECK (status IN ('received', 'defacing', 'clean', 'rejected'));

ALTER TABLE studies DROP COLUMN IF EXISTS source;
