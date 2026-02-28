-- +goose Up
ALTER TABLE studies
    ADD COLUMN analytics_required BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN analytics_status   TEXT    NOT NULL DEFAULT ''
        CHECK (analytics_status IN ('', 'pending', 'analyzing', 'complete', 'partial', 'failed'));

-- Extend routing_rules action enum to include require_analytics.
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
        'require_pixel_redaction',
        'require_analytics'
    ));

-- +goose Down
ALTER TABLE studies DROP COLUMN IF EXISTS analytics_required;
ALTER TABLE studies DROP COLUMN IF EXISTS analytics_status;
ALTER TABLE routing_rules DROP CONSTRAINT IF EXISTS routing_rules_action_check;
ALTER TABLE routing_rules ADD CONSTRAINT routing_rules_action_check
    CHECK (action IN (
        'route_to', 'require_defacing', 'auto_approve', 'require_qa', 'reject',
        'require_phi_scan', 'require_qc_check', 'require_bids_conversion',
        'require_classification', 'require_protocol_check', 'require_export',
        'require_pixel_redaction'
    ));
