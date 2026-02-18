-- +goose Up

CREATE TABLE digest_subscriptions (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email        TEXT NOT NULL,
    project_id   UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    frequency    TEXT NOT NULL CHECK (frequency IN ('weekly', 'monthly')),
    enabled      BOOLEAN NOT NULL DEFAULT TRUE,
    last_sent_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(email, project_id)
);

CREATE INDEX idx_digest_subscriptions_project ON digest_subscriptions(project_id);

-- +goose Down

DROP TABLE IF EXISTS digest_subscriptions;
