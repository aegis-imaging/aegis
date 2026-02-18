package model

import (
	"context"
	"database/sql"
	"time"
)

// DigestSubscription records a request to receive periodic summary emails for a project.
type DigestSubscription struct {
	ID          string     `json:"id"`
	Email       string     `json:"email"`
	ProjectID   string     `json:"project_id"`
	ProjectName string     `json:"project_name,omitempty"` // joined, not stored
	Frequency   string     `json:"frequency"`              // weekly | monthly
	Enabled     bool       `json:"enabled"`
	LastSentAt  *time.Time `json:"last_sent_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func CreateDigestSubscription(ctx context.Context, db *sql.DB, s *DigestSubscription) error {
	return db.QueryRowContext(ctx, `
		INSERT INTO digest_subscriptions (email, project_id, frequency, enabled)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at`,
		s.Email, s.ProjectID, s.Frequency, s.Enabled,
	).Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt)
}

func GetDigestSubscriptionByID(ctx context.Context, db *sql.DB, id string) (*DigestSubscription, error) {
	var s DigestSubscription
	err := db.QueryRowContext(ctx, `
		SELECT ds.id, ds.email, ds.project_id, p.name,
		       ds.frequency, ds.enabled, ds.last_sent_at, ds.created_at, ds.updated_at
		FROM digest_subscriptions ds
		JOIN projects p ON p.id = ds.project_id
		WHERE ds.id = $1`, id).
		Scan(&s.ID, &s.Email, &s.ProjectID, &s.ProjectName,
			&s.Frequency, &s.Enabled, &s.LastSentAt, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func ListDigestSubscriptionsByProject(ctx context.Context, db *sql.DB, projectID string) ([]DigestSubscription, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT ds.id, ds.email, ds.project_id, p.name,
		       ds.frequency, ds.enabled, ds.last_sent_at, ds.created_at, ds.updated_at
		FROM digest_subscriptions ds
		JOIN projects p ON p.id = ds.project_id
		WHERE ds.project_id = $1
		ORDER BY ds.email`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []DigestSubscription
	for rows.Next() {
		var s DigestSubscription
		if err := rows.Scan(&s.ID, &s.Email, &s.ProjectID, &s.ProjectName,
			&s.Frequency, &s.Enabled, &s.LastSentAt, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// ListAllDigestSubscriptions returns all subscriptions with project names, for the admin UI.
func ListAllDigestSubscriptions(ctx context.Context, db *sql.DB) ([]DigestSubscription, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT ds.id, ds.email, ds.project_id, p.name,
		       ds.frequency, ds.enabled, ds.last_sent_at, ds.created_at, ds.updated_at
		FROM digest_subscriptions ds
		JOIN projects p ON p.id = ds.project_id
		ORDER BY p.name, ds.email`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []DigestSubscription
	for rows.Next() {
		var s DigestSubscription
		if err := rows.Scan(&s.ID, &s.Email, &s.ProjectID, &s.ProjectName,
			&s.Frequency, &s.Enabled, &s.LastSentAt, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func DeleteDigestSubscription(ctx context.Context, db *sql.DB, id string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM digest_subscriptions WHERE id = $1`, id)
	return err
}

// ListDueSubscriptions returns enabled subscriptions whose digest is due to be sent.
// Weekly: not sent in the last 7 days.
// Monthly: not sent in the last 30 days.
func ListDueSubscriptions(ctx context.Context, db *sql.DB) ([]DigestSubscription, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT ds.id, ds.email, ds.project_id, p.name,
		       ds.frequency, ds.enabled, ds.last_sent_at, ds.created_at, ds.updated_at
		FROM digest_subscriptions ds
		JOIN projects p ON p.id = ds.project_id
		WHERE ds.enabled = TRUE
		  AND (
		    (ds.frequency = 'weekly'  AND (ds.last_sent_at IS NULL OR ds.last_sent_at < now() - INTERVAL '7 days'))
		 OR (ds.frequency = 'monthly' AND (ds.last_sent_at IS NULL OR ds.last_sent_at < now() - INTERVAL '30 days'))
		  )
		ORDER BY p.name, ds.email`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []DigestSubscription
	for rows.Next() {
		var s DigestSubscription
		if err := rows.Scan(&s.ID, &s.Email, &s.ProjectID, &s.ProjectName,
			&s.Frequency, &s.Enabled, &s.LastSentAt, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// UpdateDigestLastSent sets last_sent_at = now() for a subscription.
func UpdateDigestLastSent(ctx context.Context, db *sql.DB, id string) error {
	_, err := db.ExecContext(ctx,
		`UPDATE digest_subscriptions SET last_sent_at = now(), updated_at = now() WHERE id = $1`, id)
	return err
}

// DigestStats holds aggregate study counts for a project over a time window.
type DigestStats struct {
	ProjectID    string
	ProjectName  string
	Received     int
	Approved     int
	Rejected     int
	Pending      int
	SharesCreated int
}

// GetDigestStats queries study and share counts for a project since a given time.
func GetDigestStats(ctx context.Context, db *sql.DB, projectID string, since time.Time) (DigestStats, error) {
	var stats DigestStats
	stats.ProjectID = projectID

	err := db.QueryRowContext(ctx, `
		SELECT
		  COUNT(*) FILTER (WHERE created_at >= $2)                         AS received,
		  COUNT(*) FILTER (WHERE status = 'approved' AND updated_at >= $2) AS approved,
		  COUNT(*) FILTER (WHERE status = 'rejected' AND updated_at >= $2) AS rejected,
		  COUNT(*) FILTER (WHERE status NOT IN ('approved','rejected'))     AS pending
		FROM studies
		WHERE project_id = $1`, projectID, since).
		Scan(&stats.Received, &stats.Approved, &stats.Rejected, &stats.Pending)
	if err != nil {
		return stats, err
	}

	err = db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM export_shares es
		JOIN studies s ON s.id = es.study_id
		WHERE s.project_id = $1 AND es.created_at >= $2`,
		projectID, since).Scan(&stats.SharesCreated)
	if err != nil {
		return stats, err
	}

	return stats, nil
}
