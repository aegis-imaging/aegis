-- +goose Up
CREATE TABLE IF NOT EXISTS review_checklist_items (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    label       TEXT NOT NULL DEFAULT '',
    sort_order  INT NOT NULL DEFAULT 0,
    required    BOOLEAN NOT NULL DEFAULT true,
    enabled     BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_review_checklist_items_project ON review_checklist_items(project_id);

CREATE TABLE IF NOT EXISTS study_checklist_responses (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    study_id        UUID NOT NULL REFERENCES studies(id) ON DELETE CASCADE,
    checklist_item_id UUID NOT NULL REFERENCES review_checklist_items(id) ON DELETE CASCADE,
    checked         BOOLEAN NOT NULL DEFAULT false,
    checked_by      TEXT NOT NULL DEFAULT '',
    checked_at      TIMESTAMPTZ,
    UNIQUE(study_id, checklist_item_id)
);
CREATE INDEX IF NOT EXISTS idx_study_checklist_responses_study ON study_checklist_responses(study_id);

-- +goose Down
DROP TABLE IF EXISTS study_checklist_responses;
DROP TABLE IF EXISTS review_checklist_items;
