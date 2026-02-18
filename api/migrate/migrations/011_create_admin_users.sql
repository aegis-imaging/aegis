-- +goose Up
CREATE TABLE admin_users (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email      TEXT NOT NULL UNIQUE,
    name       TEXT NOT NULL DEFAULT '',
    role       TEXT NOT NULL CHECK (role IN ('admin', 'viewer')) DEFAULT 'admin',
    enabled    BOOLEAN NOT NULL DEFAULT TRUE,
    notes      TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE admin_users;
