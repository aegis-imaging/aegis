-- +goose Up
CREATE TABLE IF NOT EXISTS institution_contacts (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id UUID NOT NULL REFERENCES institutions(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    email      TEXT NOT NULL DEFAULT '',
    phone      TEXT NOT NULL DEFAULT '',
    role       TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_institution_contacts_inst ON institution_contacts(institution_id);

-- +goose Down
DROP TABLE IF EXISTS institution_contacts;
