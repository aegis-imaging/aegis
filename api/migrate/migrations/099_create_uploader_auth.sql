-- +goose Up
-- Native password authentication for outside data contributors.
--
-- Before this migration, AEGIS auth was entirely header-based (GCP IAP, Azure
-- Easy Auth, AWS ALB+Cognito) or API-key Bearer tokens. None of those work
-- for an outside hospital coordinator who has no Google account and no admin
-- credentials. This migration adds a native login path scoped to a single
-- function: contributing studies to projects an admin/researcher has invited
-- them to.
--
-- The model:
--   1. Researcher/admin invites by email -> uploader_invites row + magic link
--   2. Uploader clicks link, sets password -> admin_users.password_hash filled,
--      role=uploader, and a project_members row (role=uploader) is created.
--   3. Uploader logs in -> uploader_sessions row, id stored in HttpOnly cookie.
--   4. Uploader hits /api/upload/* -> middleware checks cookie, handler verifies
--      project_members membership for the target project.
--
-- Uploaders cannot log into the admin dashboard: cookie is scoped to the
-- upload-portal origin and the auth middleware in admin dashboards reads
-- header-based identity, not the uploader_sessions table.

ALTER TABLE admin_users ADD COLUMN IF NOT EXISTS password_hash TEXT;

ALTER TABLE admin_users DROP CONSTRAINT IF EXISTS admin_users_role_check;
ALTER TABLE admin_users
    ADD CONSTRAINT admin_users_role_check
        CHECK (role IN ('admin', 'viewer', 'researcher', 'uploader'));

ALTER TABLE project_members DROP CONSTRAINT IF EXISTS project_members_role_check;
ALTER TABLE project_members
    ADD CONSTRAINT project_members_role_check
        CHECK (role IN ('owner', 'coordinator', 'reviewer', 'site_coordinator', 'site_viewer', 'uploader'));

CREATE TABLE IF NOT EXISTS uploader_invites (
    id               UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    email            TEXT         NOT NULL,
    name             TEXT         NOT NULL DEFAULT '',
    project_id       UUID         NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    institution_id   UUID         REFERENCES institutions(id) ON DELETE SET NULL,
    invite_token     TEXT         NOT NULL UNIQUE,
    invited_by       TEXT         NOT NULL DEFAULT '',
    expires_at       TIMESTAMPTZ  NOT NULL,
    redeemed_at      TIMESTAMPTZ,
    redeemed_user_id UUID         REFERENCES admin_users(id) ON DELETE SET NULL,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_uploader_invites_email   ON uploader_invites(LOWER(email));
CREATE INDEX IF NOT EXISTS idx_uploader_invites_project ON uploader_invites(project_id);
CREATE INDEX IF NOT EXISTS idx_uploader_invites_token   ON uploader_invites(invite_token);

CREATE TABLE IF NOT EXISTS uploader_sessions (
    id           TEXT         PRIMARY KEY,
    user_id      UUID         NOT NULL REFERENCES admin_users(id) ON DELETE CASCADE,
    ip_address   TEXT         NOT NULL DEFAULT '',
    user_agent   TEXT         NOT NULL DEFAULT '',
    expires_at   TIMESTAMPTZ  NOT NULL,
    last_used_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_uploader_sessions_user    ON uploader_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_uploader_sessions_expires ON uploader_sessions(expires_at);

-- +goose Down
DROP TABLE IF EXISTS uploader_sessions;
DROP TABLE IF EXISTS uploader_invites;

ALTER TABLE project_members DROP CONSTRAINT IF EXISTS project_members_role_check;
ALTER TABLE project_members
    ADD CONSTRAINT project_members_role_check
        CHECK (role IN ('owner', 'coordinator', 'reviewer', 'site_coordinator', 'site_viewer'));

ALTER TABLE admin_users DROP CONSTRAINT IF EXISTS admin_users_role_check;
ALTER TABLE admin_users
    ADD CONSTRAINT admin_users_role_check
        CHECK (role IN ('admin', 'viewer', 'researcher'));

ALTER TABLE admin_users DROP COLUMN IF EXISTS password_hash;
