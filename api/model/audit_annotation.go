package model

import (
	"context"
	"database/sql"
	"time"
)

type AuditAnnotation struct {
	ID        string    `json:"id"`
	AuditID   string    `json:"audit_id"`
	Author    string    `json:"author"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

func CreateAuditAnnotation(ctx context.Context, db *sql.DB, a *AuditAnnotation) error {
	return db.QueryRowContext(ctx, `
		INSERT INTO audit_annotations (audit_id, author, body)
		VALUES ($1, $2, $3)
		RETURNING id, created_at`,
		a.AuditID, a.Author, a.Body).
		Scan(&a.ID, &a.CreatedAt)
}

func ListAuditAnnotations(ctx context.Context, db *sql.DB, auditID string) ([]AuditAnnotation, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, audit_id, author, body, created_at
		FROM audit_annotations
		WHERE audit_id = $1
		ORDER BY created_at ASC`, auditID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AuditAnnotation
	for rows.Next() {
		var a AuditAnnotation
		if err := rows.Scan(&a.ID, &a.AuditID, &a.Author, &a.Body, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func DeleteAuditAnnotation(ctx context.Context, db *sql.DB, id string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM audit_annotations WHERE id = $1`, id)
	return err
}
