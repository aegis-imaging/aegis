package model

import (
	"context"
	"database/sql"
	"time"
)

type RoutingRuleChangelogEntry struct {
	ID        string    `json:"id"`
	RuleID    string    `json:"rule_id"`
	ChangedBy string    `json:"changed_by"`
	Field     string    `json:"field"`
	OldValue  string    `json:"old_value"`
	NewValue  string    `json:"new_value"`
	CreatedAt time.Time `json:"created_at"`
}

func CreateRoutingRuleChangelogEntry(ctx context.Context, db *sql.DB, e *RoutingRuleChangelogEntry) error {
	return db.QueryRowContext(ctx, `
		INSERT INTO routing_rule_changelog (rule_id, changed_by, field, old_value, new_value)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at`,
		e.RuleID, e.ChangedBy, e.Field, e.OldValue, e.NewValue).
		Scan(&e.ID, &e.CreatedAt)
}

func ListRoutingRuleChangelog(ctx context.Context, db *sql.DB, ruleID string, limit int) ([]RoutingRuleChangelogEntry, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := db.QueryContext(ctx, `
		SELECT id, rule_id, changed_by, field, old_value, new_value, created_at
		FROM routing_rule_changelog
		WHERE rule_id = $1
		ORDER BY created_at DESC
		LIMIT $2`, ruleID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []RoutingRuleChangelogEntry
	for rows.Next() {
		var e RoutingRuleChangelogEntry
		if err := rows.Scan(&e.ID, &e.RuleID, &e.ChangedBy, &e.Field, &e.OldValue, &e.NewValue, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
