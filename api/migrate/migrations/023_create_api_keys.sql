-- +goose Up
CREATE TABLE api_keys (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT        NOT NULL CHECK (length(trim(name)) > 0),
    key_hash    TEXT        NOT NULL UNIQUE,  -- SHA-256 hex of the raw key
    key_prefix  TEXT        NOT NULL,         -- first 8 chars of raw key for display
    created_by  TEXT        NOT NULL DEFAULT '',
    enabled     BOOLEAN     NOT NULL DEFAULT TRUE,
    last_used_at TIMESTAMPTZ,
    expires_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX api_keys_key_hash_idx ON api_keys (key_hash);

-- +goose Down
DROP TABLE IF EXISTS api_keys;
