-- +goose Up
-- Webhook subscriptions: external URLs that receive POST notifications on study events.
CREATE TABLE webhook_subscriptions (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID REFERENCES projects(id) ON DELETE CASCADE,  -- NULL = all projects
    url        TEXT NOT NULL,
    events     JSONB NOT NULL DEFAULT '[]',   -- e.g. '["study.approved","study.rejected"]'
    secret     TEXT NOT NULL DEFAULT '',        -- HMAC-SHA256 signing key (empty = unsigned)
    enabled    BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_webhook_subs_project ON webhook_subscriptions(project_id);
CREATE INDEX idx_webhook_subs_enabled ON webhook_subscriptions(enabled);

-- +goose Down
DROP TABLE IF EXISTS webhook_subscriptions;
