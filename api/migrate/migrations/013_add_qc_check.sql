-- +goose Up

-- Add QC check fields to studies.
ALTER TABLE studies
    ADD COLUMN qc_required BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN qc_status   TEXT    NOT NULL DEFAULT ''
        CHECK (qc_status IN ('', 'pending', 'checking', 'pass', 'warn', 'fail', 'failed'));

-- Extend routing_rules action enum to include require_qc_check.
ALTER TABLE routing_rules DROP CONSTRAINT routing_rules_action_check;
ALTER TABLE routing_rules ADD CONSTRAINT routing_rules_action_check
    CHECK (action IN (
        'route_to',
        'require_defacing',
        'auto_approve',
        'require_qa',
        'reject',
        'require_phi_scan',
        'require_qc_check'
    ));

-- +goose Down
ALTER TABLE routing_rules DROP CONSTRAINT routing_rules_action_check;
ALTER TABLE routing_rules ADD CONSTRAINT routing_rules_action_check
    CHECK (action IN (
        'route_to',
        'require_defacing',
        'auto_approve',
        'require_qa',
        'reject',
        'require_phi_scan'
    ));

ALTER TABLE studies
    DROP COLUMN qc_status,
    DROP COLUMN qc_required;
