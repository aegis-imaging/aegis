-- +goose Up
CREATE TABLE IF NOT EXISTS audit_bookmarks (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES admin_users(id) ON DELETE CASCADE,
    audit_id   UUID NOT NULL,
    note       TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(user_id, audit_id)
);
CREATE INDEX IF NOT EXISTS idx_audit_bookmarks_user ON audit_bookmarks(user_id);

-- +goose Down
DROP TABLE IF EXISTS audit_bookmarks;
