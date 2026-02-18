-- +goose Up
CREATE TABLE audit_trail (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    action        TEXT NOT NULL,
    actor         TEXT NOT NULL DEFAULT '',
    resource_type TEXT NOT NULL DEFAULT '',
    resource_id   TEXT NOT NULL DEFAULT '',
    detail        JSONB,
    ip_address    TEXT NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_audit_trail_action ON audit_trail (action);
CREATE INDEX idx_audit_trail_created ON audit_trail (created_at);
CREATE INDEX idx_audit_trail_resource ON audit_trail (resource_type, resource_id);

-- +goose Down
DROP TABLE audit_trail;
