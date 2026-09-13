-- +goose Up
CREATE TABLE IF NOT EXISTS audit_annotations (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    audit_id   UUID NOT NULL,
    author     TEXT NOT NULL,
    body       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_audit_annotations_audit ON audit_annotations(audit_id);

-- +goose Down
DROP TABLE IF EXISTS audit_annotations;
