package model

import (
	"context"
	"database/sql"
	"time"
)

type AutoShareRule struct {
	ID             string    `json:"id"`
	ProjectID      string    `json:"project_id"`
	RecipientEmail string    `json:"recipient_email"`
	ExpiryHours    int       `json:"expiry_hours"`
	Note           string    `json:"note"`
	Enabled        bool      `json:"enabled"`
	CreatedBy      string    `json:"created_by"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func CreateAutoShareRule(ctx context.Context, db *sql.DB, r *AutoShareRule) error {
	return db.QueryRowContext(ctx, `
		INSERT INTO auto_share_rules (project_id, recipient_email, expiry_hours, note, enabled, created_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at`,
		r.ProjectID, r.RecipientEmail, r.ExpiryHours, r.Note, r.Enabled, r.CreatedBy).
		Scan(&r.ID, &r.CreatedAt, &r.UpdatedAt)
}

func ListAutoShareRules(ctx context.Context, db *sql.DB, projectID string) ([]AutoShareRule, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, project_id, recipient_email, expiry_hours, note, enabled, created_by, created_at, updated_at
		FROM auto_share_rules
		WHERE project_id = $1
		ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AutoShareRule
	for rows.Next() {
		var r AutoShareRule
		if err := rows.Scan(&r.ID, &r.ProjectID, &r.RecipientEmail, &r.ExpiryHours, &r.Note, &r.Enabled, &r.CreatedBy, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func ListEnabledAutoShareRules(ctx context.Context, db *sql.DB, projectID string) ([]AutoShareRule, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, project_id, recipient_email, expiry_hours, note, enabled, created_by, created_at, updated_at
		FROM auto_share_rules
		WHERE project_id = $1 AND enabled = true
		ORDER BY created_at ASC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AutoShareRule
	for rows.Next() {
		var r AutoShareRule
		if err := rows.Scan(&r.ID, &r.ProjectID, &r.RecipientEmail, &r.ExpiryHours, &r.Note, &r.Enabled, &r.CreatedBy, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func UpdateAutoShareRule(ctx context.Context, db *sql.DB, id string, enabled bool) error {
	_, err := db.ExecContext(ctx, `
		UPDATE auto_share_rules SET enabled = $1, updated_at = now() WHERE id = $2`, enabled, id)
	return err
}

func DeleteAutoShareRule(ctx context.Context, db *sql.DB, id string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM auto_share_rules WHERE id = $1`, id)
	return err
}
