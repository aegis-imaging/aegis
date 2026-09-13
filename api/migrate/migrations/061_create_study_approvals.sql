-- +goose Up
CREATE TABLE IF NOT EXISTS study_approvals (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    study_id   UUID NOT NULL REFERENCES studies(id) ON DELETE CASCADE,
    approved_by TEXT NOT NULL,
    action     TEXT NOT NULL DEFAULT 'approved',  -- approved, rejected
    reason     TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_study_approvals_study ON study_approvals(study_id);

-- +goose Down
DROP TABLE IF EXISTS study_approvals;
