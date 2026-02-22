// Package retention runs a background worker that enforces per-project study retention policies.
// When a project has retention_days set, approved studies older than that threshold are
// marked as "expired". The worker runs once on startup, then every 24 hours.
package retention

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/aegis-imaging/aegis/api/model"
)

// Start launches the retention background worker.
// It runs one pass immediately, then repeats every 24 hours.
// The goroutine exits when ctx is cancelled.
func Start(ctx context.Context, db *sql.DB) {
	go func() {
		run(ctx, db)
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				run(ctx, db)
			}
		}
	}()
}

// run performs one retention sweep across all projects with a policy set.
func run(ctx context.Context, db *sql.DB) {
	projects, err := model.ProjectsWithRetentionPolicy(ctx, db)
	if err != nil {
		log.Printf("retention: list projects: %v", err)
		return
	}
	for _, p := range projects {
		if p.RetentionDays == nil {
			continue
		}
		n, err := model.ExpireStudiesByRetention(ctx, db, p.ID, *p.RetentionDays)
		if err != nil {
			log.Printf("retention: project %s (%s): %v", p.Slug, p.ID, err)
			continue
		}
		if n > 0 {
			log.Printf("retention: project %s — expired %d studies (policy: %d days)", p.Slug, n, *p.RetentionDays)
		}
	}
}
