package model

import (
	"context"
	"database/sql"
	"time"
)

type RoutingRuleTemplate struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Action      string    `json:"action"`
	Modality    *string   `json:"modality,omitempty"`
	BodyPart    *string   `json:"body_part,omitempty"`
	Source      *string   `json:"source,omitempty"`
	CreatedBy   string    `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
}

func CreateRoutingRuleTemplate(ctx context.Context, db *sql.DB, t *RoutingRuleTemplate) error {
	return db.QueryRowContext(ctx, `
		INSERT INTO routing_rule_templates (name, description, action, modality, body_part, source, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at`,
		t.Name, t.Description, t.Action, t.Modality, t.BodyPart, t.Source, t.CreatedBy).
		Scan(&t.ID, &t.CreatedAt)
}

func ListRoutingRuleTemplates(ctx context.Context, db *sql.DB) ([]RoutingRuleTemplate, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, name, description, action, modality, body_part, source, created_by, created_at
		FROM routing_rule_templates
		ORDER BY name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []RoutingRuleTemplate
	for rows.Next() {
		var t RoutingRuleTemplate
		if err := rows.Scan(&t.ID, &t.Name, &t.Description, &t.Action, &t.Modality, &t.BodyPart, &t.Source, &t.CreatedBy, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func GetRoutingRuleTemplate(ctx context.Context, db *sql.DB, id string) (*RoutingRuleTemplate, error) {
	var t RoutingRuleTemplate
	err := db.QueryRowContext(ctx, `
		SELECT id, name, description, action, modality, body_part, source, created_by, created_at
		FROM routing_rule_templates WHERE id = $1`, id).
		Scan(&t.ID, &t.Name, &t.Description, &t.Action, &t.Modality, &t.BodyPart, &t.Source, &t.CreatedBy, &t.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func DeleteRoutingRuleTemplate(ctx context.Context, db *sql.DB, id string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM routing_rule_templates WHERE id = $1`, id)
	return err
}
