-- +goose Up
-- Add 'researcher' as a valid admin_users role.
-- researcher = project-scoped user (PI, site coordinator, reviewer, etc.)
-- Their access is determined entirely by project_members entries, not platform-wide.
-- Existing 'admin' and 'viewer' roles are unchanged.
ALTER TABLE admin_users
    DROP CONSTRAINT IF EXISTS admin_users_role_check;
ALTER TABLE admin_users
    ADD CONSTRAINT admin_users_role_check
        CHECK (role IN ('admin', 'viewer', 'researcher'));

-- +goose Down
-- Revert: remove any researcher users first (or this will fail on constraint violation)
UPDATE admin_users SET role = 'viewer' WHERE role = 'researcher';
ALTER TABLE admin_users
    DROP CONSTRAINT IF EXISTS admin_users_role_check;
ALTER TABLE admin_users
    ADD CONSTRAINT admin_users_role_check
        CHECK (role IN ('admin', 'viewer'));
