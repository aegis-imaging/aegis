-- +goose Up
ALTER TABLE projects ADD COLUMN stuck_threshold_minutes INT;

-- +goose Down
ALTER TABLE projects DROP COLUMN stuck_threshold_minutes;
