-- +goose Up

-- Time-limited, revocable export shares for outbound study sharing.
-- The raw token is never stored; only its SHA-256 hex hash is kept.
CREATE TABLE export_shares (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    study_id        UUID NOT NULL REFERENCES studies(id),
    token_hash      TEXT NOT NULL UNIQUE,
    recipient_email TEXT NOT NULL,
    note            TEXT NOT NULL DEFAULT '',
    expires_at      TIMESTAMPTZ NOT NULL,
    created_by      TEXT NOT NULL DEFAULT 'admin',
    revoked_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_export_shares_study ON export_shares (study_id);
CREATE INDEX idx_export_shares_token ON export_shares (token_hash);

-- Immutable log of every time a share token is redeemed.
CREATE TABLE export_downloads (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    share_id    UUID NOT NULL REFERENCES export_shares(id),
    client_ip   TEXT NOT NULL DEFAULT '',
    accessed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE export_downloads;
DROP TABLE export_shares;
