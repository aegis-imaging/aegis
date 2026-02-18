-- +goose Up
ALTER TABLE studies
    ADD COLUMN bids_required BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN bids_status   TEXT    NOT NULL DEFAULT ''
        CHECK (bids_status IN ('', 'pending', 'converting', 'complete', 'failed'));

-- Extend routing_rules action to include require_bids_conversion
ALTER TABLE routing_rules DROP CONSTRAINT routing_rules_action_check;
ALTER TABLE routing_rules ADD CONSTRAINT routing_rules_action_check
    CHECK (action IN ('route_to','require_defacing','auto_approve','require_qa','reject','require_phi_scan','require_qc_check','require_bids_conversion'));

-- +goose Down
ALTER TABLE studies DROP COLUMN bids_required;
ALTER TABLE studies DROP COLUMN bids_status;

ALTER TABLE routing_rules DROP CONSTRAINT routing_rules_action_check;
ALTER TABLE routing_rules ADD CONSTRAINT routing_rules_action_check
    CHECK (action IN ('route_to','require_defacing','auto_approve','require_qa','reject','require_phi_scan','require_qc_check'));
