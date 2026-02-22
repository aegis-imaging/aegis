-- +goose Up
-- Add 'expired' to the studies status check constraint for retention policy support.
ALTER TABLE studies DROP CONSTRAINT IF EXISTS studies_status_check;
ALTER TABLE studies ADD CONSTRAINT studies_status_check
    CHECK (status IN ('received', 'defacing', 'defaced', 'clean', 'approved', 'rejected', 'expired'));

-- +goose Down
ALTER TABLE studies DROP CONSTRAINT IF EXISTS studies_status_check;
ALTER TABLE studies ADD CONSTRAINT studies_status_check
    CHECK (status IN ('received', 'defacing', 'defaced', 'clean', 'approved', 'rejected'));
