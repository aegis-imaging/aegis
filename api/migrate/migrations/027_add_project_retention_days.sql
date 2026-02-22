-- +goose Up
-- retention_days: if NULL, studies are kept indefinitely.
-- A background worker soft-deletes (or marks) approved studies older than this threshold.
ALTER TABLE projects ADD COLUMN retention_days INTEGER;

-- +goose Down
ALTER TABLE projects DROP COLUMN retention_days;
