-- +goose Up

-- Add burned-in PHI scan fields to studies.
ALTER TABLE studies
    ADD COLUMN phi_scan_required BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN phi_scan_status   TEXT    NOT NULL DEFAULT ''
        CHECK (phi_scan_status IN ('', 'pending', 'scanning', 'clean', 'flagged', 'failed'));

-- Extend routing_rules action enum to include require_phi_scan.
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

-- +goose Down
ALTER TABLE routing_rules DROP CONSTRAINT routing_rules_action_check;
ALTER TABLE routing_rules ADD CONSTRAINT routing_rules_action_check
    CHECK (action IN (
        'route_to',
        'require_defacing',
        'auto_approve',
        'require_qa',
        'reject'
    ));

ALTER TABLE studies
    DROP COLUMN phi_scan_status,
    DROP COLUMN phi_scan_required;
