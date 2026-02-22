-- +goose Up
-- Add optional rejection reason to studies.
-- Populated when an admin rejects a study; shown in the dashboard and included
-- in uploader notification emails.
ALTER TABLE studies ADD COLUMN rejection_reason TEXT;

-- +goose Down
ALTER TABLE studies DROP COLUMN IF EXISTS rejection_reason;
