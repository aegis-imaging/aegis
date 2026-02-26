package model

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
)

type SavedView struct {
	ID        string          `json:"id"`
	UserID    string          `json:"user_id"`
	Name      string          `json:"name"`
	Filters   json.RawMessage `json:"filters"`
	Shared    bool            `json:"shared"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

func CreateSavedView(ctx context.Context, db *sql.DB, v *SavedView) error {
	return db.QueryRowContext(ctx, `
		INSERT INTO saved_views (user_id, name, filters, shared)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at`,
		v.UserID, v.Name, v.Filters, v.Shared).
		Scan(&v.ID, &v.CreatedAt, &v.UpdatedAt)
}

func ListSavedViews(ctx context.Context, db *sql.DB, userID string) ([]SavedView, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, user_id, name, filters, shared, created_at, updated_at
		FROM saved_views
		WHERE user_id = $1 OR shared = true
		ORDER BY name ASC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SavedView
	for rows.Next() {
		var v SavedView
		if err := rows.Scan(&v.ID, &v.UserID, &v.Name, &v.Filters, &v.Shared, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func DeleteSavedView(ctx context.Context, db *sql.DB, id, userID string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM saved_views WHERE id = $1 AND user_id = $2`, id, userID)
	return err
}

func UpdateSavedView(ctx context.Context, db *sql.DB, id, userID, name string, filters json.RawMessage, shared bool) error {
	_, err := db.ExecContext(ctx, `
		UPDATE saved_views SET name = $1, filters = $2, shared = $3, updated_at = now()
		WHERE id = $4 AND user_id = $5`, name, filters, shared, id, userID)
	return err
}
