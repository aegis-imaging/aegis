-- +goose Up
-- Add study_size_bytes to track total DICOM file size for a study.
ALTER TABLE studies ADD COLUMN study_size_bytes BIGINT NOT NULL DEFAULT 0;
