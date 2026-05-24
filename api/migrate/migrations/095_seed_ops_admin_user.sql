-- +goose Up
--
-- Seed ops@aegisimaging.ai as an admin user.
--
-- Context: the first_admin bootstrap (api/main.go) only seeds ONE email at
-- API startup (read from var.first_admin_email in terraform tfvars). In
-- production that's matthewsenjem@gmail.com. But ops@aegisimaging.ai is
-- also a primary operator account (see memory) and needs admin access too.
--
-- This migration is idempotent (ON CONFLICT DO NOTHING). It runs once per
-- fresh DB, including after the Cloud SQL replacement on 2026-05-24 that
-- emptied the prior admin_users table.
--
-- If the human ever changes which two accounts are primary, edit this
-- migration's seeded rows (or add a follow-up migration); don't expect
-- bootstrap to manage multiple emails.

INSERT INTO admin_users (email, name, role, enabled, notes)
VALUES (
    'ops@aegisimaging.ai',
    'Matthew Senjem (ops)',
    'admin',
    TRUE,
    'Primary ops account — M365-hosted email + Google account. Seeded via migration 095 after the 2026-05-24 SQL recreation.'
)
ON CONFLICT (email) DO NOTHING;

-- +goose Down
DELETE FROM admin_users WHERE email = 'ops@aegisimaging.ai';
