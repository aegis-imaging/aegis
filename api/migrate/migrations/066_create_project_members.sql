-- +goose Up
-- project_members: links admin users to projects with a specific role.
-- This is the foundation of project-level access control (clinical trial model).
--
-- Coordinating center roles (institution_id IS NULL):
--   owner       - Principal Investigator; full project control
--   coordinator - Data manager / study coordinator; write access to studies
--   reviewer    - QC reviewer / statistician; read all + approve/reject
--
-- Site roles (institution_id IS NOT NULL = their participating site):
--   site_coordinator - Site research coordinator; uploads + views own site's data
--   site_viewer      - Site monitor / sponsor; read-only for own site's data
--
-- When institution_id IS NULL  → user sees ALL studies in the project
-- When institution_id IS NOT NULL → user sees ONLY studies where study.institution_id matches
CREATE TABLE project_members (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id      UUID        NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    admin_user_id   UUID        NOT NULL REFERENCES admin_users(id) ON DELETE CASCADE,
    role            TEXT        NOT NULL CHECK (role IN ('owner', 'coordinator', 'reviewer', 'site_coordinator', 'site_viewer')),
    institution_id  UUID        REFERENCES institutions(id) ON DELETE SET NULL,
    notes           TEXT        NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(project_id, admin_user_id)
);

CREATE INDEX idx_project_members_project      ON project_members(project_id);
CREATE INDEX idx_project_members_admin_user   ON project_members(admin_user_id);
CREATE INDEX idx_project_members_institution  ON project_members(institution_id);

-- +goose Down
DROP TABLE IF EXISTS project_members;
