-- +goose Up
ALTER TABLE studies ADD COLUMN auto_share_url TEXT;

-- +goose Down
ALTER TABLE studies DROP COLUMN auto_share_url;
