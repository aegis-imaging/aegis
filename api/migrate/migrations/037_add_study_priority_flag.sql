-- +goose Up
ALTER TABLE studies ADD COLUMN priority_flag BOOLEAN NOT NULL DEFAULT false;
CREATE INDEX idx_studies_priority_flag ON studies(priority_flag) WHERE priority_flag = true;

-- +goose Down
DROP INDEX IF EXISTS idx_studies_priority_flag;
ALTER TABLE studies DROP COLUMN IF EXISTS priority_flag;
