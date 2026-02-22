-- +goose Up
ALTER TABLE export_shares ADD COLUMN max_downloads INT;

-- +goose Down
ALTER TABLE export_shares DROP COLUMN IF EXISTS max_downloads;
