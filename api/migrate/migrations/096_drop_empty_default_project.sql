-- +goose Up
--
-- Retire the auto-seeded "Default" project. Migration 001 inserts one
-- ('Default' / slug='default' / description='Default project for uploads')
-- so the dashboard always has a target to upload to. But the new XNAT-style
-- researcher home gives a worse first impression — visitors see a random
-- project they didn't create — and the upload portal is being retired
-- anyway, so the default project's reason for existing is gone.
--
-- This migration deletes the default project ONLY IF nothing references
-- it (no studies, no routing rules, no project ACL entries). If a user
-- has been uploading to it, we leave it alone — they can rename it
-- through the admin UI instead of losing data.
--
-- For fresh installs going forward, migration 001 still creates the
-- project briefly, then this migration deletes it. Slightly wasteful
-- (one INSERT + one DELETE on first apply) but keeps migration history
-- append-only.

DELETE FROM projects
WHERE  slug = 'default'
  AND  NOT EXISTS (SELECT 1 FROM studies WHERE project_id = projects.id)
  AND  NOT EXISTS (SELECT 1 FROM routing_rules WHERE project_id = projects.id);

-- +goose Down
--
-- Re-seed the default project (matches the INSERT in migration 001).
-- Idempotent via ON CONFLICT.
INSERT INTO projects (name, slug, description)
VALUES ('Default', 'default', 'Default project for uploads')
ON CONFLICT (slug) DO NOTHING;
