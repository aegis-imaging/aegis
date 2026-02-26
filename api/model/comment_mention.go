package model

import (
	"context"
	"database/sql"
	"time"
)

type CommentMention struct {
	ID        string    `json:"id"`
	CommentID string    `json:"comment_id"`
	UserEmail string    `json:"user_email"`
	CreatedAt time.Time `json:"created_at"`
}

func CreateCommentMention(ctx context.Context, db *sql.DB, commentID, userEmail string) (*CommentMention, error) {
	var m CommentMention
	err := db.QueryRowContext(ctx, `
		INSERT INTO comment_mentions (comment_id, user_email)
		VALUES ($1, $2)
		RETURNING id, comment_id, user_email, created_at`,
		commentID, userEmail).
		Scan(&m.ID, &m.CommentID, &m.UserEmail, &m.CreatedAt)
	return &m, err
}

func ListCommentMentions(ctx context.Context, db *sql.DB, commentID string) ([]CommentMention, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, comment_id, user_email, created_at
		FROM comment_mentions
		WHERE comment_id = $1
		ORDER BY created_at ASC`, commentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CommentMention
	for rows.Next() {
		var m CommentMention
		if err := rows.Scan(&m.ID, &m.CommentID, &m.UserEmail, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// ListMentionsForUser returns all mentions for a specific user across all comments.
func ListMentionsForUser(ctx context.Context, db *sql.DB, userEmail string, limit int) ([]CommentMention, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := db.QueryContext(ctx, `
		SELECT id, comment_id, user_email, created_at
		FROM comment_mentions
		WHERE user_email = $1
		ORDER BY created_at DESC
		LIMIT $2`, userEmail, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CommentMention
	for rows.Next() {
		var m CommentMention
		if err := rows.Scan(&m.ID, &m.CommentID, &m.UserEmail, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
