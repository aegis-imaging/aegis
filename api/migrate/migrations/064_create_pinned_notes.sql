-- +goose Up
CREATE TABLE IF NOT EXISTS study_pinned_notes (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    study_id   UUID NOT NULL REFERENCES studies(id) ON DELETE CASCADE,
    author     TEXT NOT NULL,
    body       TEXT NOT NULL,
    pinned     BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_study_pinned_notes_study ON study_pinned_notes(study_id);

-- +goose Down
DROP TABLE IF EXISTS study_pinned_notes;
