-- +goose Up
ALTER TABLE studies ADD COLUMN IF NOT EXISTS assigned_to UUID REFERENCES admin_users(id) ON DELETE SET NULL;
ALTER TABLE studies ADD COLUMN IF NOT EXISTS assigned_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE studies DROP COLUMN IF EXISTS assigned_at;
ALTER TABLE studies DROP COLUMN IF EXISTS assigned_to;
