-- +goose Up
-- Extend the studies.source CHECK constraint to allow 'spoke' for studies that
-- entered AEGIS via an on-prem AEGIS Router with verified mTLS client cert.
-- Also extend routing_rules.source so routing rules can match spoke-sourced
-- traffic explicitly.
ALTER TABLE studies DROP CONSTRAINT IF EXISTS studies_source_check;
ALTER TABLE studies ADD CONSTRAINT studies_source_check
    CHECK (source IN ('external', 'internal', 'spoke'));

ALTER TABLE routing_rules DROP CONSTRAINT IF EXISTS routing_rules_source_check;
ALTER TABLE routing_rules ADD CONSTRAINT routing_rules_source_check
    CHECK (source IS NULL OR source IN ('external', 'internal', 'spoke'));

-- +goose Down
ALTER TABLE studies DROP CONSTRAINT IF EXISTS studies_source_check;
ALTER TABLE studies ADD CONSTRAINT studies_source_check
    CHECK (source IN ('external', 'internal'));

ALTER TABLE routing_rules DROP CONSTRAINT IF EXISTS routing_rules_source_check;
ALTER TABLE routing_rules ADD CONSTRAINT routing_rules_source_check
    CHECK (source IS NULL OR source IN ('external', 'internal'));
