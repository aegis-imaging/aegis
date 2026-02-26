-- +goose Up
CREATE TABLE IF NOT EXISTS study_transfers (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    study_id        UUID NOT NULL REFERENCES studies(id) ON DELETE CASCADE,
    from_project_id UUID NOT NULL REFERENCES projects(id),
    to_project_id   UUID NOT NULL REFERENCES projects(id),
    transferred_by  TEXT NOT NULL,
    reason          TEXT NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_study_transfers_study ON study_transfers(study_id);

-- +goose Down
DROP TABLE IF EXISTS study_transfers;
