package model

import (
	"context"
	"database/sql"
	"time"
)

type AdminSession struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
	CreatedAt time.Time `json:"created_at"`
}

func RecordAdminSession(ctx context.Context, db *sql.DB, userID, ip, ua string) (*AdminSession, error) {
	var s AdminSession
	err := db.QueryRowContext(ctx, `
		INSERT INTO admin_sessions (user_id, ip_address, user_agent)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, ip_address, user_agent, created_at`,
		userID, ip, ua).
		Scan(&s.ID, &s.UserID, &s.IPAddress, &s.UserAgent, &s.CreatedAt)
	return &s, err
}

func ListAdminSessions(ctx context.Context, db *sql.DB, userID string, limit int) ([]AdminSession, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := db.QueryContext(ctx, `
		SELECT id, user_id, ip_address, user_agent, created_at
		FROM admin_sessions
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AdminSession
	for rows.Next() {
		var s AdminSession
		if err := rows.Scan(&s.ID, &s.UserID, &s.IPAddress, &s.UserAgent, &s.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
