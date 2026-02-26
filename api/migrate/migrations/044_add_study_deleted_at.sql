-- +goose Up
ALTER TABLE studies ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE studies DROP COLUMN IF EXISTS deleted_at;
