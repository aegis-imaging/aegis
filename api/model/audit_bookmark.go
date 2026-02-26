package model

import (
	"context"
	"database/sql"
	"time"
)

type AuditBookmark struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	AuditID   string    `json:"audit_id"`
	Note      string    `json:"note"`
	CreatedAt time.Time `json:"created_at"`
}

func CreateAuditBookmark(ctx context.Context, db *sql.DB, b *AuditBookmark) error {
	return db.QueryRowContext(ctx, `
		INSERT INTO audit_bookmarks (user_id, audit_id, note)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, audit_id) DO NOTHING
		RETURNING id, created_at`,
		b.UserID, b.AuditID, b.Note).
		Scan(&b.ID, &b.CreatedAt)
}

func ListAuditBookmarks(ctx context.Context, db *sql.DB, userID string, limit int) ([]AuditBookmark, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := db.QueryContext(ctx, `
		SELECT id, user_id, audit_id, note, created_at
		FROM audit_bookmarks
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AuditBookmark
	for rows.Next() {
		var b AuditBookmark
		if err := rows.Scan(&b.ID, &b.UserID, &b.AuditID, &b.Note, &b.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func DeleteAuditBookmark(ctx context.Context, db *sql.DB, id, userID string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM audit_bookmarks WHERE id = $1 AND user_id = $2`, id, userID)
	return err
}
