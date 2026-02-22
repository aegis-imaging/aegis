-- +goose Up
-- webhook_deliveries: immutable log of every HTTP delivery attempt.
CREATE TABLE webhook_deliveries (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subscription_id UUID NOT NULL REFERENCES webhook_subscriptions(id) ON DELETE CASCADE,
    event           TEXT NOT NULL,
    url             TEXT NOT NULL,
    attempt         INT NOT NULL DEFAULT 1,
    status_code     INT,                  -- NULL on network/timeout error
    success         BOOLEAN NOT NULL,
    error_message   TEXT,
    delivered_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX webhook_deliveries_sub_idx ON webhook_deliveries(subscription_id, delivered_at DESC);

-- +goose Down
DROP TABLE IF EXISTS webhook_deliveries;
