-- +goose Up
-- Tracks the last time an SLA stuck-alert was fired for each study.
-- Used by the SLA scheduler to avoid re-alerting on every hourly tick.
CREATE TABLE study_sla_alerts (
    study_id       UUID PRIMARY KEY REFERENCES studies(id) ON DELETE CASCADE,
    last_alerted_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS study_sla_alerts;
