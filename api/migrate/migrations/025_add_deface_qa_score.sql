-- +goose Up
ALTER TABLE studies ADD COLUMN deface_qa_score NUMERIC(5,4);

-- +goose Down
ALTER TABLE studies DROP COLUMN IF EXISTS deface_qa_score;
