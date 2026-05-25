-- +goose Up
-- Adds an optional institution_id link to desktop_installer_invites so the
-- per-institution detail panel (/admin/institutions/:id) can list invites
-- scoped to one institution alongside its satellites + browser upload
-- allowlist. Existing invites (sent before this column existed) keep
-- institution_id NULL and stay visible in the global invite list only.
--
-- ON DELETE SET NULL — when an institution is removed we keep the audit
-- trail of who was invited; the invite simply detaches from the
-- now-gone institution.
ALTER TABLE desktop_installer_invites
    ADD COLUMN IF NOT EXISTS institution_id UUID
        REFERENCES institutions(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_installer_invites_institution
    ON desktop_installer_invites (institution_id, sent_at DESC)
    WHERE institution_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_installer_invites_institution;
ALTER TABLE desktop_installer_invites DROP COLUMN IF EXISTS institution_id;
