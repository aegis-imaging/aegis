-- +goose Up
-- restricted: when true, only project_members AND platform admins can see this project.
-- When false (default), platform admin/viewer roles see the project as before (backwards-compat).
-- Researcher-role users always obey the membership check regardless of this flag.
ALTER TABLE projects
    ADD COLUMN restricted BOOLEAN NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE projects DROP COLUMN IF EXISTS restricted;
