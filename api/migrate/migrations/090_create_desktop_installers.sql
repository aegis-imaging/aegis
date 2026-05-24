-- +goose Up
-- Desktop client installer registry + per-recipient install invites.
--
-- desktop_installers: one row per (product, platform, version) binary.
-- The binary itself lives in the existing Storage backend under
-- 'installers/{product}/{version}/{filename}' (or an external URL when
-- the installer isn't hosted by AEGIS, e.g. GitHub Releases).
--
-- desktop_installer_invites: one row per emailed download link. Carries
-- both the recipient-facing access token (in the download URL) and a
-- single-use pairing_token the app exchanges for a long-lived API key.

CREATE TABLE desktop_installers (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    product       TEXT        NOT NULL CHECK (product IN ('uploader', 'dimse-bridge')),
    platform      TEXT        NOT NULL,
    version       TEXT        NOT NULL,
    storage_key   TEXT,                 -- NULL when external_url is set
    external_url  TEXT,                 -- when set, skip storage download and link directly
    filename      TEXT        NOT NULL,
    size_bytes    BIGINT      NOT NULL DEFAULT 0,
    sha256        TEXT        NOT NULL DEFAULT '',
    changelog     TEXT,
    is_current    BOOLEAN     NOT NULL DEFAULT FALSE,
    released_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by    TEXT        NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (product, platform, version),
    CHECK (storage_key IS NOT NULL OR external_url IS NOT NULL)
);

-- Partial unique: at most one is_current per (product, platform).
CREATE UNIQUE INDEX idx_installers_current_unique
    ON desktop_installers (product, platform)
    WHERE is_current;

CREATE INDEX idx_installers_product_platform
    ON desktop_installers (product, platform, released_at DESC);

CREATE TABLE desktop_installer_invites (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    installer_id      UUID        NOT NULL REFERENCES desktop_installers(id) ON DELETE CASCADE,
    recipient_email   TEXT        NOT NULL,
    recipient_name    TEXT,
    token             TEXT        NOT NULL UNIQUE,         -- URL-safe random, in /install/{token}
    pairing_token     TEXT        NOT NULL UNIQUE,         -- single-use, exchanged via /api/install/pair
    expires_at        TIMESTAMPTZ NOT NULL,
    sent_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    sent_by           TEXT        NOT NULL,
    first_clicked_at  TIMESTAMPTZ,
    last_clicked_at   TIMESTAMPTZ,
    click_count       INT         NOT NULL DEFAULT 0,
    paired_at         TIMESTAMPTZ,
    paired_api_key_id UUID                                 -- references api_keys(id); SET NULL on key delete
                       REFERENCES api_keys(id) ON DELETE SET NULL
);

CREATE INDEX idx_installer_invites_email
    ON desktop_installer_invites (recipient_email, sent_at DESC);

CREATE INDEX idx_installer_invites_installer
    ON desktop_installer_invites (installer_id, sent_at DESC);

-- +goose Down
DROP TABLE IF EXISTS desktop_installer_invites;
DROP TABLE IF EXISTS desktop_installers;
