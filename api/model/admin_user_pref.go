package model

import (
	"context"
	"database/sql"
	"time"

	"github.com/lib/pq"
)

// AdminUserPreferences stores per-user notification and digest preferences.
type AdminUserPreferences struct {
	AdminUserID     string    `json:"admin_user_id"`
	DigestFrequency string    `json:"digest_frequency"` // "none" | "daily" | "weekly" | "monthly"
	NotifyEvents    []string  `json:"notify_events"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// ValidDigestFrequencies is the set of accepted values for DigestFrequency.
var ValidDigestFrequencies = map[string]bool{
	"none":    true,
	"daily":   true,
	"weekly":  true,
	"monthly": true,
}

// ValidNotifyEvents is the set of accepted values for NotifyEvents.
var ValidNotifyEvents = map[string]bool{
	"study.stuck":        true,
	"pipeline.failed":    true,
	"study.phi_flagged":  true,
	"study.approved":     true,
	"study.rejected":     true,
}

// GetAdminUserPreferences returns preferences for a user; returns defaults if none exist.
func GetAdminUserPreferences(ctx context.Context, db *sql.DB, adminUserID string) (*AdminUserPreferences, error) {
	p := &AdminUserPreferences{
		AdminUserID:     adminUserID,
		DigestFrequency: "weekly",
		NotifyEvents:    []string{},
		UpdatedAt:       time.Now(),
	}
	err := db.QueryRowContext(ctx, `
		SELECT admin_user_id, digest_frequency, notify_events, updated_at
		FROM admin_user_preferences
		WHERE admin_user_id = $1`, adminUserID).
		Scan(&p.AdminUserID, &p.DigestFrequency, pq.Array(&p.NotifyEvents), &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return p, nil // return defaults
	}
	if err != nil {
		return nil, err
	}
	if p.NotifyEvents == nil {
		p.NotifyEvents = []string{}
	}
	return p, nil
}

// UpsertAdminUserPreferences creates or updates preferences for a user.
func UpsertAdminUserPreferences(ctx context.Context, db *sql.DB, p *AdminUserPreferences) error {
	_, err := db.ExecContext(ctx, `
		INSERT INTO admin_user_preferences (admin_user_id, digest_frequency, notify_events, updated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (admin_user_id) DO UPDATE
		SET digest_frequency = EXCLUDED.digest_frequency,
		    notify_events    = EXCLUDED.notify_events,
		    updated_at       = NOW()`,
		p.AdminUserID, p.DigestFrequency, pq.Array(p.NotifyEvents))
	return err
}
