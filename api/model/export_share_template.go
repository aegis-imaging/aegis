package model

import (
	"context"
	"database/sql"
	"time"
)

type ExportShareTemplate struct {
	ID             string    `json:"id"`
	ProjectID      *string   `json:"project_id,omitempty"`
	Name           string    `json:"name"`
	RecipientEmail string    `json:"recipient_email"`
	Note           string    `json:"note"`
	ExpiryHours    int       `json:"expiry_hours"`
	CreatedBy      string    `json:"created_by"`
	CreatedAt      time.Time `json:"created_at"`
}

func CreateExportShareTemplate(ctx context.Context, db *sql.DB, t *ExportShareTemplate) error {
	return db.QueryRowContext(ctx, `
		INSERT INTO export_share_templates (project_id, name, recipient_email, note, expiry_hours, created_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at`,
		t.ProjectID, t.Name, t.RecipientEmail, t.Note, t.ExpiryHours, t.CreatedBy).
		Scan(&t.ID, &t.CreatedAt)
}

func ListExportShareTemplates(ctx context.Context, db *sql.DB, projectID string) ([]ExportShareTemplate, error) {
	var rows *sql.Rows
	var err error
	if projectID != "" {
		rows, err = db.QueryContext(ctx, `
			SELECT id, project_id, name, recipient_email, note, expiry_hours, created_by, created_at
			FROM export_share_templates
			WHERE project_id = $1 OR project_id IS NULL
			ORDER BY name ASC`, projectID)
	} else {
		rows, err = db.QueryContext(ctx, `
			SELECT id, project_id, name, recipient_email, note, expiry_hours, created_by, created_at
			FROM export_share_templates
			ORDER BY name ASC`)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ExportShareTemplate
	for rows.Next() {
		var t ExportShareTemplate
		if err := rows.Scan(&t.ID, &t.ProjectID, &t.Name, &t.RecipientEmail, &t.Note, &t.ExpiryHours, &t.CreatedBy, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func DeleteExportShareTemplate(ctx context.Context, db *sql.DB, id string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM export_share_templates WHERE id = $1`, id)
	return err
}
