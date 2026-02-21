-- +goose Up

-- Add 'defaced' to the status state machine.
-- UpdateStudyDefaced() sets status='defaced' after successful defacing.
ALTER TABLE studies DROP CONSTRAINT IF EXISTS studies_status_check;
ALTER TABLE studies ADD CONSTRAINT studies_status_check
    CHECK (status IN ('received', 'defacing', 'defaced', 'clean', 'approved', 'rejected'));

-- +goose Down
ALTER TABLE studies DROP CONSTRAINT IF EXISTS studies_status_check;
ALTER TABLE studies ADD CONSTRAINT studies_status_check
    CHECK (status IN ('received', 'defacing', 'clean', 'approved', 'rejected'));
