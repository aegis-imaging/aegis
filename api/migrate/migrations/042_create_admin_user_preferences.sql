-- +goose Up
CREATE TABLE admin_user_preferences (
    admin_user_id    UUID        PRIMARY KEY REFERENCES admin_users(id) ON DELETE CASCADE,
    digest_frequency TEXT        NOT NULL DEFAULT 'weekly',
    notify_events    TEXT[]      NOT NULL DEFAULT '{}',
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS admin_user_preferences;
