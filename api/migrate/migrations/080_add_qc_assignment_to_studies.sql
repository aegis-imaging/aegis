-- +goose Up
-- QC assignment: which analyst (if any) is currently responsible for this
-- study's QC review. Separate from the existing study_assignments table
-- (which is for peer review of clinical interpretation) — qc_assigned_to is
-- specifically the image-analyst QC workflow.
ALTER TABLE studies
    ADD COLUMN IF NOT EXISTS qc_assigned_to       UUID REFERENCES admin_users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS qc_assigned_at       TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS qc_review_started_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS qc_review_ended_at   TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_studies_qc_assigned_open
    ON studies (qc_assigned_to)
    WHERE qc_review_ended_at IS NULL AND qc_assigned_to IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_studies_qc_assigned_open;
ALTER TABLE studies
    DROP COLUMN IF EXISTS qc_review_ended_at,
    DROP COLUMN IF EXISTS qc_review_started_at,
    DROP COLUMN IF EXISTS qc_assigned_at,
    DROP COLUMN IF EXISTS qc_assigned_to;
