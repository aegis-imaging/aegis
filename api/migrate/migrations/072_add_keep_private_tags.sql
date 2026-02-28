-- +goose Up
-- Add keep_private_tags boolean to anonymization profiles.
-- When true, vendor-specific private DICOM tags (odd group numbers) are
-- retained during client-side de-identification. Important for DTI,
-- ASL, and other sequences where acquisition parameters live in
-- Siemens CSA headers, GE private tags, or Philips private tags.

ALTER TABLE anon_profiles
    ADD COLUMN keep_private_tags BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down
ALTER TABLE anon_profiles DROP COLUMN IF EXISTS keep_private_tags;
