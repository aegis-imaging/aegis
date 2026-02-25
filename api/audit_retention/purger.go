// Package audit_retention runs a background worker that deletes audit_trail rows
// older than AUDIT_RETENTION_DAYS. Disabled when AUDIT_RETENTION_DAYS is 0 (default).
// The worker runs once on startup, then repeats every 24 hours.
package audit_retention

import (
	"context"
	"database/sql"
	"log"
	"time"
)

// Start launches the audit-log purge background worker.
// It is a no-op when retentionDays is 0.
// The goroutine exits when ctx is cancelled.
func Start(ctx context.Context, db *sql.DB, retentionDays int) {
	if retentionDays <= 0 {
		return
	}
	go func() {
		purge(ctx, db, retentionDays)
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				purge(ctx, db, retentionDays)
			}
		}
	}()
}

// purge deletes audit_trail rows older than retentionDays.
func purge(ctx context.Context, db *sql.DB, retentionDays int) {
	cutoff := time.Now().UTC().AddDate(0, 0, -retentionDays)
	result, err := db.ExecContext(ctx,
		`DELETE FROM audit_trail WHERE created_at < $1`, cutoff)
	if err != nil {
		log.Printf("audit_retention: purge: %v", err)
		return
	}
	n, _ := result.RowsAffected()
	if n > 0 {
		log.Printf("audit_retention: purged %d rows older than %d days (cutoff %s)",
			n, retentionDays, cutoff.Format(time.RFC3339))
	}
}
