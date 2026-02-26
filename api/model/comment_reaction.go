package model

import (
	"context"
	"database/sql"
	"time"
)

type CommentReaction struct {
	ID        string    `json:"id"`
	CommentID string    `json:"comment_id"`
	UserID    string    `json:"user_id"`
	Emoji     string    `json:"emoji"`
	CreatedAt time.Time `json:"created_at"`
}

func AddCommentReaction(ctx context.Context, db *sql.DB, commentID, userID, emoji string) (*CommentReaction, error) {
	var r CommentReaction
	err := db.QueryRowContext(ctx, `
		INSERT INTO comment_reactions (comment_id, user_id, emoji)
		VALUES ($1, $2, $3)
		ON CONFLICT (comment_id, user_id, emoji) DO UPDATE SET comment_id = comment_reactions.comment_id
		RETURNING id, comment_id, user_id, emoji, created_at`,
		commentID, userID, emoji).
		Scan(&r.ID, &r.CommentID, &r.UserID, &r.Emoji, &r.CreatedAt)
	return &r, err
}

func ListCommentReactions(ctx context.Context, db *sql.DB, commentID string) ([]CommentReaction, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, comment_id, user_id, emoji, created_at
		FROM comment_reactions
		WHERE comment_id = $1
		ORDER BY created_at ASC`, commentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CommentReaction
	for rows.Next() {
		var r CommentReaction
		if err := rows.Scan(&r.ID, &r.CommentID, &r.UserID, &r.Emoji, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func DeleteCommentReaction(ctx context.Context, db *sql.DB, commentID, userID, emoji string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM comment_reactions WHERE comment_id = $1 AND user_id = $2 AND emoji = $3`,
		commentID, userID, emoji)
	return err
}
