-- +goose Up
ALTER TABLE studies
    ADD COLUMN pixel_redaction_required BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN pixel_redaction_status   TEXT    NOT NULL DEFAULT ''
        CHECK (pixel_redaction_status IN ('', 'pending', 'redacting', 'complete', 'failed'));

-- Extend routing_rules action enum to include require_pixel_redaction.
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
        'require_protocol_check',
        'require_export',
        'require_pixel_redaction'
    ));

-- +goose Down
ALTER TABLE studies DROP COLUMN IF EXISTS pixel_redaction_required;
ALTER TABLE studies DROP COLUMN IF EXISTS pixel_redaction_status;
ALTER TABLE routing_rules DROP CONSTRAINT IF EXISTS routing_rules_action_check;
ALTER TABLE routing_rules ADD CONSTRAINT routing_rules_action_check
    CHECK (action IN (
        'route_to', 'require_defacing', 'auto_approve', 'require_qa', 'reject',
        'require_phi_scan', 'require_qc_check', 'require_bids_conversion',
        'require_classification', 'require_protocol_check', 'require_export'
    ));
