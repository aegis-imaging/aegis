-- +goose Up
CREATE TABLE invite_requests (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name          TEXT        NOT NULL,
    email         TEXT        NOT NULL,
    org           TEXT        NOT NULL DEFAULT '',
    message       TEXT        NOT NULL DEFAULT '',
    status        TEXT        NOT NULL DEFAULT 'pending', -- pending | approved | denied
    ip            TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reviewed_at   TIMESTAMPTZ,
    reviewed_by   TEXT,
    invite_code_id UUID       REFERENCES invite_codes(id) ON DELETE SET NULL
);

CREATE INDEX invite_requests_status_idx ON invite_requests (status);
CREATE INDEX invite_requests_email_idx  ON invite_requests (email);

-- +goose Down
DROP TABLE IF EXISTS invite_requests;
