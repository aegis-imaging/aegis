-- +goose Up
CREATE TABLE IF NOT EXISTS routing_rule_changelog (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_id    UUID NOT NULL REFERENCES routing_rules(id) ON DELETE CASCADE,
    changed_by TEXT NOT NULL,
    field      TEXT NOT NULL,
    old_value  TEXT NOT NULL DEFAULT '',
    new_value  TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_routing_rule_changelog_rule ON routing_rule_changelog(rule_id);

-- +goose Down
DROP TABLE IF EXISTS routing_rule_changelog;
