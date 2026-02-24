-- +goose Up
CREATE TABLE study_relationships (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    study_id         UUID        NOT NULL REFERENCES studies(id) ON DELETE CASCADE,
    related_study_id UUID        NOT NULL REFERENCES studies(id) ON DELETE CASCADE,
    relationship     TEXT        NOT NULL,
    notes            TEXT,
    created_by       TEXT        NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(study_id, related_study_id)
);
CREATE INDEX idx_study_relationships_study_id   ON study_relationships(study_id);
CREATE INDEX idx_study_relationships_related_id ON study_relationships(related_study_id);

-- +goose Down
DROP TABLE IF EXISTS study_relationships;
