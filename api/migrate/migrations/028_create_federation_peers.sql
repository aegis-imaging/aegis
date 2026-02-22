-- +goose Up
-- federation_peers: trusted remote AEGIS instances that can pull approved studies.
-- This is a stub for future cross-tenant federation. No data flows yet.
CREATE TABLE federation_peers (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    slug        TEXT NOT NULL UNIQUE,
    api_url     TEXT NOT NULL,
    api_key_hash TEXT,               -- SHA-256 of the bearer token we send them (stored hashed)
    enabled     BOOLEAN NOT NULL DEFAULT true,
    notes       TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS federation_peers;
