-- +goose Up
ALTER TABLE projects ADD COLUMN storage_quota_bytes BIGINT;

-- +goose Down
ALTER TABLE projects DROP COLUMN storage_quota_bytes;
