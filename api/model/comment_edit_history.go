package model

import (
	"context"
	"database/sql"
	"time"
)

type CommentEditHistory struct {
	ID        string    `json:"id"`
	CommentID string    `json:"comment_id"`
	OldBody   string    `json:"old_body"`
	NewBody   string    `json:"new_body"`
	EditedBy  string    `json:"edited_by"`
	CreatedAt time.Time `json:"created_at"`
}

func CreateCommentEditHistory(ctx context.Context, db *sql.DB, h *CommentEditHistory) error {
	return db.QueryRowContext(ctx, `
		INSERT INTO comment_edit_history (comment_id, old_body, new_body, edited_by)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at`,
		h.CommentID, h.OldBody, h.NewBody, h.EditedBy).
		Scan(&h.ID, &h.CreatedAt)
}

func ListCommentEditHistory(ctx context.Context, db *sql.DB, commentID string) ([]CommentEditHistory, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, comment_id, old_body, new_body, edited_by, created_at
		FROM comment_edit_history
		WHERE comment_id = $1
		ORDER BY created_at DESC`, commentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CommentEditHistory
	for rows.Next() {
		var h CommentEditHistory
		if err := rows.Scan(&h.ID, &h.CommentID, &h.OldBody, &h.NewBody, &h.EditedBy, &h.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}
