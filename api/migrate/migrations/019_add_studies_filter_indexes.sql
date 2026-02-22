-- +goose Up
-- Functional indexes for case-insensitive exact-match filters on the studies
-- list endpoint (GET /api/studies?modality=&body_part=). Without these the
-- query falls back to a full-table scan via idx_studies_status and then
-- filters in memory, which degrades at scale.
CREATE INDEX IF NOT EXISTS idx_studies_modality
    ON studies (upper(modality));

CREATE INDEX IF NOT EXISTS idx_studies_body_part
    ON studies (upper(body_part));

-- Composite functional index accelerates the common combined filter:
-- status + modality (e.g. "all received MRI studies").
CREATE INDEX IF NOT EXISTS idx_studies_status_modality
    ON studies (status, upper(modality));

-- +goose Down
DROP INDEX IF EXISTS idx_studies_status_modality;
DROP INDEX IF EXISTS idx_studies_body_part;
DROP INDEX IF EXISTS idx_studies_modality;
