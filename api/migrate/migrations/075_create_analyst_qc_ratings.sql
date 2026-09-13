-- +goose Up

CREATE TABLE IF NOT EXISTS analyst_qc_ratings (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    study_id         UUID NOT NULL REFERENCES studies(id) ON DELETE CASCADE,
    analyst_id       UUID NOT NULL REFERENCES admin_users(id),
    rating_quality   SMALLINT NOT NULL CHECK (rating_quality BETWEEN 0 AND 4),
    rating_motion    SMALLINT NOT NULL CHECK (rating_motion BETWEEN 0 AND 4),
    rating_snr       SMALLINT CHECK (rating_snr BETWEEN 0 AND 4),
    rating_coverage  SMALLINT CHECK (rating_coverage BETWEEN 0 AND 4),
    rating_artifacts SMALLINT CHECK (rating_artifacts BETWEEN 0 AND 4),
    rating_overall   SMALLINT NOT NULL CHECK (rating_overall BETWEEN 0 AND 4),
    comments         TEXT NOT NULL DEFAULT '',
    review_type      TEXT NOT NULL DEFAULT 'standard'
        CHECK (review_type IN ('standard', 'defacing', 'analytics', 'protocol')),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_qc_ratings_study ON analyst_qc_ratings(study_id);
CREATE INDEX IF NOT EXISTS idx_qc_ratings_analyst ON analyst_qc_ratings(analyst_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_qc_ratings_unique
    ON analyst_qc_ratings(study_id, analyst_id, review_type);

-- +goose Down

DROP TABLE IF EXISTS analyst_qc_ratings;
