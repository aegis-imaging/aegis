-- +goose Up
-- Invite codes for landing page access control.
-- Each code is unique, per-person, and can be individually revoked.
-- Codes are validated via the public POST /api/invite/validate endpoint
-- so the token is never baked into the client bundle.

CREATE TABLE invite_codes (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    code       TEXT        NOT NULL UNIQUE,
    label      TEXT        NOT NULL DEFAULT '',
    enabled    BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    used_at    TIMESTAMPTZ,
    used_by_ip TEXT
);

CREATE INDEX idx_invite_codes_code ON invite_codes (code);

-- +goose Down
DROP TABLE IF EXISTS invite_codes;
