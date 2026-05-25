-- +goose Up
-- Per-institution allowlist of browser upload methods. Default behavior
-- when no row exists for (institution_id, method_id) is ALLOWED — this
-- table is additive in the deny direction. Existing institutions see no
-- change until an admin explicitly adds a deny row.
--
-- Method IDs are managed in code (see api/model/institution_upload_allowlist.go
-- KnownUploadMethods). Storing them as TEXT rather than a Postgres enum
-- keeps the migration story simple as we add methods over time.
--
-- enabled = TRUE rows are technically redundant (absence-of-row already
-- means allowed) but we keep them so admins can attach a note explaining
-- why a method is intentionally on for a given institution — e.g. "site
-- IRB authorized 2026-05 for direct DIMSE pull".
CREATE TABLE IF NOT EXISTS institution_upload_allowlist (
    institution_id  UUID        NOT NULL REFERENCES institutions(id) ON DELETE CASCADE,
    method_id       TEXT        NOT NULL,
    enabled         BOOLEAN     NOT NULL DEFAULT TRUE,
    note            TEXT        NOT NULL DEFAULT '',
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by      TEXT        NOT NULL,
    PRIMARY KEY (institution_id, method_id)
);

-- +goose Down
DROP TABLE IF EXISTS institution_upload_allowlist;
