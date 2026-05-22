-- +goose Up
CREATE TABLE IF NOT EXISTS destination_maintenance_windows (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    destination_id UUID NOT NULL REFERENCES destinations(id) ON DELETE CASCADE,
    reason         TEXT NOT NULL DEFAULT '',
    starts_at      TIMESTAMPTZ NOT NULL,
    ends_at        TIMESTAMPTZ NOT NULL,
    created_by     TEXT NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_dest_maint_dest ON destination_maintenance_windows(destination_id);

-- +goose Down
DROP TABLE IF EXISTS destination_maintenance_windows;
