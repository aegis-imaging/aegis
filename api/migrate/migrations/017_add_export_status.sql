-- +goose Up
ALTER TABLE studies
    ADD COLUMN export_required BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN export_status   TEXT    NOT NULL DEFAULT ''
        CHECK (export_status IN ('', 'pending', 'exporting', 'exported', 'failed'));

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
        'require_export'
    ));

-- +goose Down
ALTER TABLE studies DROP COLUMN export_required, DROP COLUMN export_status;

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
