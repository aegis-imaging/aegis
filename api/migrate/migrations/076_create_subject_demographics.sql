-- +goose Up

CREATE TABLE IF NOT EXISTS subject_demographics (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subject_id       TEXT NOT NULL,
    project_id       UUID NOT NULL REFERENCES projects(id),
    sex              TEXT NOT NULL DEFAULT '' CHECK (sex IN ('', 'M', 'F', 'NB', 'O')),
    age_at_scan      INT,
    diagnosis        TEXT NOT NULL DEFAULT '',
    education_years  SMALLINT,
    mmse_score       SMALLINT CHECK (mmse_score IS NULL OR mmse_score BETWEEN 0 AND 30),
    moca_score       SMALLINT CHECK (moca_score IS NULL OR moca_score BETWEEN 0 AND 30),
    cdr_global       NUMERIC(3,1) CHECK (cdr_global IS NULL OR cdr_global IN (0, 0.5, 1, 2, 3)),
    apoe_genotype    TEXT DEFAULT '',
    notes            TEXT NOT NULL DEFAULT '',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(subject_id, project_id)
);

CREATE INDEX IF NOT EXISTS idx_subj_demo_subject ON subject_demographics(subject_id);
CREATE INDEX IF NOT EXISTS idx_subj_demo_project ON subject_demographics(project_id);

-- +goose Down

DROP TABLE IF EXISTS subject_demographics;
