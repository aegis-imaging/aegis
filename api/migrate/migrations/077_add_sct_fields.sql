-- +goose Up
ALTER TABLE studies ADD COLUMN sct_required BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE studies ADD COLUMN sct_status VARCHAR(32) NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE studies DROP COLUMN sct_required;
ALTER TABLE studies DROP COLUMN sct_status;
