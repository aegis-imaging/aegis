-- +goose Up
-- Adds mTLS client certificate identity to institutions so spoke-site routers
-- can authenticate by presenting their per-spoke client cert. The cloud
-- middleware computes the cert's SHA-256 thumbprint and matches it here.
ALTER TABLE institutions
    ADD COLUMN IF NOT EXISTS client_cert_thumbprint TEXT,
    ADD COLUMN IF NOT EXISTS client_cert_subject_dn TEXT,
    ADD COLUMN IF NOT EXISTS client_cert_enrolled_at TIMESTAMPTZ;

CREATE UNIQUE INDEX IF NOT EXISTS idx_institutions_client_cert_thumbprint
    ON institutions (client_cert_thumbprint)
    WHERE client_cert_thumbprint IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_institutions_client_cert_thumbprint;
ALTER TABLE institutions
    DROP COLUMN IF EXISTS client_cert_enrolled_at,
    DROP COLUMN IF EXISTS client_cert_subject_dn,
    DROP COLUMN IF EXISTS client_cert_thumbprint;
