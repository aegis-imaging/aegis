-- +goose Up

CREATE TABLE IF NOT EXISTS roi_results (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    study_id       UUID NOT NULL REFERENCES studies(id) ON DELETE CASCADE,
    tool           TEXT NOT NULL,
    atlas_name     TEXT NOT NULL DEFAULT '',
    template_name  TEXT NOT NULL DEFAULT '',
    roi_number     INT NOT NULL DEFAULT 0,
    roi_name       TEXT NOT NULL,
    metric_type    TEXT NOT NULL,
    metric_value   DOUBLE PRECISION NOT NULL,
    hemisphere     TEXT NOT NULL DEFAULT '',
    scan_type      TEXT NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_roi_results_study ON roi_results(study_id);
CREATE INDEX IF NOT EXISTS idx_roi_results_roi_name ON roi_results(roi_name);
CREATE INDEX IF NOT EXISTS idx_roi_results_tool_atlas ON roi_results(tool, atlas_name);
CREATE INDEX IF NOT EXISTS idx_roi_results_metric ON roi_results(metric_type);

CREATE TABLE IF NOT EXISTS longitudinal_roi_results (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    baseline_study_id   UUID NOT NULL REFERENCES studies(id) ON DELETE CASCADE,
    followup_study_id   UUID NOT NULL REFERENCES studies(id) ON DELETE CASCADE,
    tool                TEXT NOT NULL,
    atlas_name          TEXT NOT NULL DEFAULT '',
    roi_name            TEXT NOT NULL,
    baseline_value      DOUBLE PRECISION,
    followup_value      DOUBLE PRECISION,
    change_value        DOUBLE PRECISION,
    change_percent      DOUBLE PRECISION,
    annualized_change   DOUBLE PRECISION,
    metric_type         TEXT NOT NULL,
    scan_interval_days  INT NOT NULL DEFAULT 0,
    hemisphere          TEXT NOT NULL DEFAULT '',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_long_roi_baseline ON longitudinal_roi_results(baseline_study_id);
CREATE INDEX IF NOT EXISTS idx_long_roi_followup ON longitudinal_roi_results(followup_study_id);
CREATE INDEX IF NOT EXISTS idx_long_roi_name ON longitudinal_roi_results(roi_name);

CREATE TABLE IF NOT EXISTS analytics_composite_scores (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    study_id          UUID NOT NULL REFERENCES studies(id) ON DELETE CASCADE,
    baseline_study_id UUID REFERENCES studies(id),
    tool              TEXT NOT NULL,
    score_name        TEXT NOT NULL,
    score_value       DOUBLE PRECISION NOT NULL,
    metadata          JSONB DEFAULT '{}',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_composite_study ON analytics_composite_scores(study_id);
CREATE INDEX IF NOT EXISTS idx_composite_name ON analytics_composite_scores(score_name);

-- +goose Down

DROP TABLE IF EXISTS analytics_composite_scores;
DROP TABLE IF EXISTS longitudinal_roi_results;
DROP TABLE IF EXISTS roi_results;
