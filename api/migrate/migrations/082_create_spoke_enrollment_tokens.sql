-- +goose Up
-- One-time enrollment tokens that a spoke router exchanges for a signed
-- client cert during its bootstrap (./bin/aegis-router-init --site-token=...).
-- Tokens are stored as SHA-256 hashes (never plaintext) and are tied to one
-- institution so the resulting cert is bound to that spoke at issuance.
CREATE TABLE IF NOT EXISTS spoke_enrollment_tokens (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id UUID NOT NULL REFERENCES institutions(id) ON DELETE CASCADE,
    token_hash     TEXT NOT NULL UNIQUE,
    label          TEXT NOT NULL DEFAULT '',
    created_by     TEXT NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at     TIMESTAMPTZ NOT NULL,
    used_at        TIMESTAMPTZ,
    revoked_at     TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_spoke_enrollment_tokens_institution
    ON spoke_enrollment_tokens (institution_id);
CREATE INDEX IF NOT EXISTS idx_spoke_enrollment_tokens_expires
    ON spoke_enrollment_tokens (expires_at)
    WHERE used_at IS NULL AND revoked_at IS NULL;

-- +goose Down
DROP TABLE IF EXISTS spoke_enrollment_tokens;
