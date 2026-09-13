-- +goose Up
CREATE TABLE IF NOT EXISTS saved_views (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    TEXT NOT NULL,
    name       TEXT NOT NULL,
    filters    JSONB NOT NULL DEFAULT '{}',
    shared     BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_saved_views_user ON saved_views(user_id);

-- +goose Down
DROP TABLE IF EXISTS saved_views;
