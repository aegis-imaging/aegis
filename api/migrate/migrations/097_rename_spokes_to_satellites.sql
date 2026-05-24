-- +goose Up
--
-- Rename "spokes" → "satellites" in the schema. "Spoke" was an unintuitive
-- name from the hub-and-spoke topology metaphor; "Satellite" is the
-- user-facing term we now use everywhere (admin dashboard already migrated
-- in PR #522). This migration brings the schema in line with the UX.
--
-- Renames:
--   - table:   spoke_enrollment_tokens → satellite_enrollment_tokens
--   - indexes: idx_spoke_enrollment_tokens_* → idx_satellite_enrollment_tokens_*
--   - source CHECK: 'spoke' → 'satellite' in studies.source + routing_rules.source

ALTER TABLE IF EXISTS spoke_enrollment_tokens RENAME TO satellite_enrollment_tokens;
ALTER INDEX IF EXISTS idx_spoke_enrollment_tokens_institution RENAME TO idx_satellite_enrollment_tokens_institution;
ALTER INDEX IF EXISTS idx_spoke_enrollment_tokens_expires    RENAME TO idx_satellite_enrollment_tokens_expires;

ALTER TABLE studies DROP CONSTRAINT IF EXISTS studies_source_check;
UPDATE studies SET source = 'satellite' WHERE source = 'spoke';
ALTER TABLE studies ADD CONSTRAINT studies_source_check
    CHECK (source IN ('external', 'internal', 'satellite'));

ALTER TABLE routing_rules DROP CONSTRAINT IF EXISTS routing_rules_source_check;
UPDATE routing_rules SET source = 'satellite' WHERE source = 'spoke';
ALTER TABLE routing_rules ADD CONSTRAINT routing_rules_source_check
    CHECK (source IS NULL OR source IN ('external', 'internal', 'satellite'));

-- +goose Down
ALTER TABLE routing_rules DROP CONSTRAINT IF EXISTS routing_rules_source_check;
UPDATE routing_rules SET source = 'spoke' WHERE source = 'satellite';
ALTER TABLE routing_rules ADD CONSTRAINT routing_rules_source_check
    CHECK (source IS NULL OR source IN ('external', 'internal', 'spoke'));

ALTER TABLE studies DROP CONSTRAINT IF EXISTS studies_source_check;
UPDATE studies SET source = 'spoke' WHERE source = 'satellite';
ALTER TABLE studies ADD CONSTRAINT studies_source_check
    CHECK (source IN ('external', 'internal', 'spoke'));

ALTER INDEX IF EXISTS idx_satellite_enrollment_tokens_expires    RENAME TO idx_spoke_enrollment_tokens_expires;
ALTER INDEX IF EXISTS idx_satellite_enrollment_tokens_institution RENAME TO idx_spoke_enrollment_tokens_institution;
ALTER TABLE IF EXISTS satellite_enrollment_tokens RENAME TO spoke_enrollment_tokens;
