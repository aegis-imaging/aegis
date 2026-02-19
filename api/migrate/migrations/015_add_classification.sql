-- +goose Up
-- Add metadata classification fields for smart routing.
-- Studies can arrive with empty modality/body_part; the classification service
-- reads DICOM headers and fills them in so routing rules fire correctly.

ALTER TABLE studies
    ADD COLUMN classification_required BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN classification_status   TEXT    NOT NULL DEFAULT ''
        CHECK (classification_status IN ('', 'pending', 'classifying', 'classified', 'failed'));

ALTER TABLE routing_rules DROP CONSTRAINT routing_rules_action_check;
ALTER TABLE routing_rules ADD CONSTRAINT routing_rules_action_check
    CHECK (action IN ('route_to','require_defacing','auto_approve','require_qa','reject',
                      'require_phi_scan','require_qc_check','require_bids_conversion','require_classification'));

-- +goose Down
ALTER TABLE studies DROP COLUMN classification_required, DROP COLUMN classification_status;
ALTER TABLE routing_rules DROP CONSTRAINT routing_rules_action_check;
ALTER TABLE routing_rules ADD CONSTRAINT routing_rules_action_check
    CHECK (action IN ('route_to','require_defacing','auto_approve','require_qa','reject',
                      'require_phi_scan','require_qc_check','require_bids_conversion'));
