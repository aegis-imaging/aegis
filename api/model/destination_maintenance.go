package model

import (
	"context"
	"database/sql"
	"time"
)

type DestinationMaintenanceWindow struct {
	ID            string    `json:"id"`
	DestinationID string    `json:"destination_id"`
	Reason        string    `json:"reason"`
	StartsAt      time.Time `json:"starts_at"`
	EndsAt        time.Time `json:"ends_at"`
	CreatedBy     string    `json:"created_by"`
	CreatedAt     time.Time `json:"created_at"`
}

func CreateDestinationMaintenance(ctx context.Context, db *sql.DB, m *DestinationMaintenanceWindow) error {
	return db.QueryRowContext(ctx, `
		INSERT INTO destination_maintenance_windows (destination_id, reason, starts_at, ends_at, created_by)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at`,
		m.DestinationID, m.Reason, m.StartsAt, m.EndsAt, m.CreatedBy).
		Scan(&m.ID, &m.CreatedAt)
}

func ListDestinationMaintenance(ctx context.Context, db *sql.DB, destID string) ([]DestinationMaintenanceWindow, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, destination_id, reason, starts_at, ends_at, created_by, created_at
		FROM destination_maintenance_windows
		WHERE destination_id = $1
		ORDER BY starts_at DESC`, destID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DestinationMaintenanceWindow
	for rows.Next() {
		var m DestinationMaintenanceWindow
		if err := rows.Scan(&m.ID, &m.DestinationID, &m.Reason, &m.StartsAt, &m.EndsAt, &m.CreatedBy, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func DeleteDestinationMaintenance(ctx context.Context, db *sql.DB, id, destID string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM destination_maintenance_windows WHERE id = $1 AND destination_id = $2`, id, destID)
	return err
}
