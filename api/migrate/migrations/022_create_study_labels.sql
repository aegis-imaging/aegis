CREATE TABLE study_labels (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    study_id   UUID        NOT NULL REFERENCES studies(id) ON DELETE CASCADE,
    label      TEXT        NOT NULL CHECK (length(trim(label)) > 0 AND length(label) <= 80),
    created_by TEXT        NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX study_labels_study_id_idx ON study_labels (study_id);
CREATE UNIQUE INDEX study_labels_study_label_uidx ON study_labels (study_id, lower(label));
