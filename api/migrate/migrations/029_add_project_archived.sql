-- +goose Up
ALTER TABLE projects ADD COLUMN archived BOOLEAN NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE projects DROP COLUMN IF EXISTS archived;
