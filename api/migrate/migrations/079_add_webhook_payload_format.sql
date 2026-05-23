-- +goose Up
-- Per-subscription payload format. AEGIS-envelope JSON has always been the
-- default; this lets EMR/RIS integrations subscribe with payload_format='fhir'
-- to receive a bare FHIR R4 ImagingStudy resource as the POST body instead.
ALTER TABLE webhook_subscriptions
    ADD COLUMN payload_format TEXT NOT NULL DEFAULT 'aegis'
        CHECK (payload_format IN ('aegis', 'fhir'));

-- +goose Down
ALTER TABLE webhook_subscriptions DROP COLUMN IF EXISTS payload_format;
