CREATE TABLE project_phi_config (
    project_id          UUID    PRIMARY KEY REFERENCES projects(id) ON DELETE CASCADE,
    confidence_threshold NUMERIC(4,3) NOT NULL DEFAULT 0.4
                         CHECK (confidence_threshold >= 0 AND confidence_threshold <= 1),
    min_text_length     INTEGER NOT NULL DEFAULT 3 CHECK (min_text_length >= 1),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
