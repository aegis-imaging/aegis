-- +goose Up
-- Discrete findings raised by image analysts during QC review.
-- Distinct from `analyst_qc_ratings` (which captures numeric quality scores):
-- a finding is a free-text observation tied to a specific issue
-- ("burned-in text on slice 47", "incomplete defacing in left frontal",
-- "wrong orientation marker") that downstream reviewers can act on or
-- dismiss. Multiple findings per study per analyst are expected.
CREATE TABLE IF NOT EXISTS qc_findings (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    study_id        UUID NOT NULL REFERENCES studies(id) ON DELETE CASCADE,
    analyst_id      UUID REFERENCES admin_users(id) ON DELETE SET NULL,
    analyst_email   TEXT NOT NULL DEFAULT '',
    category        TEXT NOT NULL,    -- 'phi_leak' | 'defacing' | 'motion' | 'protocol' | 'coverage' | 'other'
    severity        TEXT NOT NULL,    -- 'info' | 'minor' | 'major' | 'critical'
    body            TEXT NOT NULL,
    series_uid      TEXT NOT NULL DEFAULT '',  -- optional: scope a finding to one series
    instance_index  INTEGER,                   -- optional: scope to one instance (0-indexed)
    resolved_at     TIMESTAMPTZ,
    resolved_by     TEXT NOT NULL DEFAULT '',
    resolution_note TEXT NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_qc_findings_study     ON qc_findings (study_id);
CREATE INDEX IF NOT EXISTS idx_qc_findings_analyst   ON qc_findings (analyst_id);
CREATE INDEX IF NOT EXISTS idx_qc_findings_open      ON qc_findings (study_id) WHERE resolved_at IS NULL;

-- +goose Down
DROP TABLE IF EXISTS qc_findings;
