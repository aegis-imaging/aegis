-- +goose Up
-- Adds an optional project_id link to desktop_installer_invites so an admin
-- (or project-owner researcher) can mint a desktop-installer invite BOUND TO
-- A PROJECT. When the recipient installs + pairs, the pairing flow provisions
-- an uploader identity (admin_users role='uploader' + project_members
-- role='uploader') for that project — the desktop mirror of the browser
-- uploader invite. Existing invites (sent before this column existed) keep
-- project_id NULL and pair exactly as before (no provisioning).
--
-- ON DELETE SET NULL — when a project is removed we keep the invite history;
-- the invite simply detaches and pairs unscoped.
ALTER TABLE desktop_installer_invites
    ADD COLUMN IF NOT EXISTS project_id UUID
        REFERENCES projects(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_installer_invites_project
    ON desktop_installer_invites (project_id, sent_at DESC)
    WHERE project_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_installer_invites_project;
ALTER TABLE desktop_installer_invites DROP COLUMN IF EXISTS project_id;
