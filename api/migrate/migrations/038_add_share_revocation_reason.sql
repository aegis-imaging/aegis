-- +goose Up
ALTER TABLE export_shares ADD COLUMN revocation_reason TEXT;

-- +goose Down
ALTER TABLE export_shares DROP COLUMN IF EXISTS revocation_reason;
