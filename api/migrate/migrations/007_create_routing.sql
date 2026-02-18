-- +goose Up
-- Destinations: external DICOM endpoints studies can be forwarded to
CREATE TABLE destinations (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    slug        TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    type        TEXT NOT NULL CHECK (type IN ('dicomweb', 'dimse')),

    -- DICOMweb destination fields
    dicomweb_url         TEXT NOT NULL DEFAULT '',
    dicomweb_auth_header TEXT NOT NULL DEFAULT '', -- e.g. "Bearer <token>"; store encrypted in prod

    -- DIMSE destination fields (future)
    ae_title    TEXT NOT NULL DEFAULT '',
    host        TEXT NOT NULL DEFAULT '',
    port        INTEGER NOT NULL DEFAULT 0,

    enabled     BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_destinations_slug ON destinations (slug);
CREATE INDEX idx_destinations_enabled ON destinations (enabled);

-- Routing rules: conditions → action mappings, evaluated on study ingest
CREATE TABLE routing_rules (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    priority    INTEGER NOT NULL DEFAULT 100,   -- lower = evaluated first
    enabled     BOOLEAN NOT NULL DEFAULT TRUE,

    -- Conditions (NULL = wildcard / matches any)
    project_id  UUID REFERENCES projects (id) ON DELETE CASCADE,  -- NULL = any project
    modality    TEXT,                -- NULL = any; e.g. 'MRI', 'CT', 'PET'
    body_part   TEXT,                -- NULL = any; e.g. 'HEAD', 'CHEST'
    source      TEXT CHECK (source IN ('external', 'internal')),   -- NULL = any

    -- Action
    action          TEXT NOT NULL CHECK (action IN (
                        'route_to',          -- forward to a destination
                        'require_defacing',  -- force defacing_required=true
                        'auto_approve',      -- skip manual QC, set status=approved
                        'require_qa',        -- hold for manual QC (no-op; default behaviour)
                        'reject'             -- auto-reject
                    )),
    destination_id  UUID REFERENCES destinations (id) ON DELETE SET NULL,  -- used when action='route_to'

    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_routing_rules_priority ON routing_rules (priority, enabled);
CREATE INDEX idx_routing_rules_project  ON routing_rules (project_id);

-- Routing log: audit trail of which rules fired for which studies
CREATE TABLE routing_log (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    study_id        UUID NOT NULL REFERENCES studies (id) ON DELETE CASCADE,
    rule_id         UUID NOT NULL REFERENCES routing_rules (id) ON DELETE CASCADE,
    action          TEXT NOT NULL,
    destination_id  UUID REFERENCES destinations (id) ON DELETE SET NULL,
    outcome         TEXT NOT NULL DEFAULT '',   -- e.g. 'dispatched', 'auto_approved', 'rejected'
    detail          JSONB,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_routing_log_study ON routing_log (study_id);
CREATE INDEX idx_routing_log_rule  ON routing_log (rule_id);

-- +goose Down
DROP TABLE IF EXISTS routing_log;
DROP TABLE IF EXISTS routing_rules;
DROP TABLE IF EXISTS destinations;
