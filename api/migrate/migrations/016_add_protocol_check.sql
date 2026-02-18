-- +goose Up

-- Protocol templates: per-project, per-scanner expected acquisition parameters.
CREATE TABLE protocol_templates (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id       UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name             TEXT NOT NULL,
    description      TEXT NOT NULL DEFAULT '',
    manufacturer     TEXT NOT NULL DEFAULT '',
    model            TEXT NOT NULL DEFAULT '',
    software_version TEXT NOT NULL DEFAULT '',
    sequence_type    TEXT NOT NULL DEFAULT '',
    rules            JSONB NOT NULL DEFAULT '[]',
    enabled          BOOLEAN NOT NULL DEFAULT TRUE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_protocol_templates_project ON protocol_templates(project_id);

-- Study-level protocol compliance fields.
ALTER TABLE studies
    ADD COLUMN protocol_required BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN protocol_status   TEXT    NOT NULL DEFAULT ''
        CHECK (protocol_status IN ('', 'pending', 'checking', 'compliant', 'minor_deviations', 'non_compliant', 'failed'));

-- Extend routing_rules action enum to include require_protocol_check.
ALTER TABLE routing_rules DROP CONSTRAINT routing_rules_action_check;
ALTER TABLE routing_rules ADD CONSTRAINT routing_rules_action_check
    CHECK (action IN (
        'route_to',
        'require_defacing',
        'auto_approve',
        'require_qa',
        'reject',
        'require_phi_scan',
        'require_qc_check',
        'require_bids_conversion',
        'require_classification',
        'require_protocol_check'
    ));

-- +goose Down
ALTER TABLE studies DROP COLUMN protocol_required, DROP COLUMN protocol_status;
DROP TABLE protocol_templates;
ALTER TABLE routing_rules DROP CONSTRAINT routing_rules_action_check;
ALTER TABLE routing_rules ADD CONSTRAINT routing_rules_action_check
    CHECK (action IN (
        'route_to',
        'require_defacing',
        'auto_approve',
        'require_qa',
        'reject',
        'require_phi_scan',
        'require_qc_check',
        'require_bids_conversion',
        'require_classification'
    ));
