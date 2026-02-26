package model

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type AdminNotification struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	Link      string    `json:"link"`
	Read      bool      `json:"read"`
	CreatedAt time.Time `json:"created_at"`
}

func CreateAdminNotification(ctx context.Context, db *sql.DB, n *AdminNotification) error {
	return db.QueryRowContext(ctx, `
		INSERT INTO admin_notifications (user_id, title, body, link)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at`,
		n.UserID, n.Title, n.Body, n.Link).
		Scan(&n.ID, &n.CreatedAt)
}

func ListAdminNotifications(ctx context.Context, db *sql.DB, userID string, unreadOnly bool, limit int) ([]AdminNotification, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	where := "WHERE user_id = $1"
	if unreadOnly {
		where += " AND read = false"
	}
	rows, err := db.QueryContext(ctx, fmt.Sprintf(`
		SELECT id, user_id, title, body, link, read, created_at
		FROM admin_notifications
		%s
		ORDER BY created_at DESC
		LIMIT $2`, where), userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AdminNotification
	for rows.Next() {
		var n AdminNotification
		if err := rows.Scan(&n.ID, &n.UserID, &n.Title, &n.Body, &n.Link, &n.Read, &n.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func CountUnreadNotifications(ctx context.Context, db *sql.DB, userID string) (int, error) {
	var count int
	err := db.QueryRowContext(ctx, `SELECT count(*) FROM admin_notifications WHERE user_id = $1 AND read = false`, userID).Scan(&count)
	return count, err
}

func MarkNotificationRead(ctx context.Context, db *sql.DB, notificationID, userID string) error {
	_, err := db.ExecContext(ctx, `UPDATE admin_notifications SET read = true WHERE id = $1 AND user_id = $2`, notificationID, userID)
	return err
}

func MarkAllNotificationsRead(ctx context.Context, db *sql.DB, userID string) (int64, error) {
	res, err := db.ExecContext(ctx, `UPDATE admin_notifications SET read = true WHERE user_id = $1 AND read = false`, userID)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
