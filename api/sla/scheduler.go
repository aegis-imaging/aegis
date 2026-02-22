// Package sla provides a background scheduler that checks for studies stuck in
// the pipeline longer than a configured SLA threshold and sends alert emails.
package sla

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/aegis-imaging/aegis/api/email"
	"github.com/aegis-imaging/aegis/api/model"
)

// Start launches a background goroutine that checks for stuck studies once per
// hour and sends alert emails. It is a no-op when minutes == 0 (disabled) or
// when alertEmail is empty. The goroutine stops when ctx is cancelled.
func Start(ctx context.Context, db *sql.DB, mailer *email.Client, minutes, cooldownHours int, alertEmail string) {
	if minutes <= 0 || alertEmail == "" {
		return
	}
	go run(ctx, db, mailer, minutes, cooldownHours, alertEmail)
}

func run(ctx context.Context, db *sql.DB, mailer *email.Client, minutes, cooldownHours int, alertEmail string) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	// Run once on startup so operators see stuck studies quickly after deploy.
	check(ctx, db, mailer, minutes, cooldownHours, alertEmail)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			check(ctx, db, mailer, minutes, cooldownHours, alertEmail)
		}
	}
}

func check(ctx context.Context, db *sql.DB, mailer *email.Client, minutes, cooldownHours int, alertEmail string) {
	studies, err := model.GetUnalertedStuckStudies(ctx, db, minutes, cooldownHours)
	if err != nil {
		log.Printf("sla: query stuck studies: %v", err)
		return
	}
	if len(studies) == 0 {
		return
	}

	stuckList := make([]email.StuckStudy, len(studies))
	for i, s := range studies {
		stuckList[i] = email.StuckStudy{
			StudyInstanceUID: s.StudyInstanceUID,
			Status:          s.Status,
			UpdatedAt:       s.UpdatedAt,
		}
	}

	subject, body := email.StudiesStuck(stuckList, minutes)
	if err := mailer.Send(ctx, alertEmail, subject, body); err != nil {
		log.Printf("sla: send alert to %s: %v", alertEmail, err)
		return
	}
	log.Printf("sla: sent alert for %d stuck studies to %s", len(studies), alertEmail)

	// Mark all alerted studies so we don't re-alert within the cooldown window.
	for _, s := range studies {
		if err := model.MarkStudySLAAlerted(ctx, db, s.ID); err != nil {
			log.Printf("sla: mark alerted for study %s: %v", s.ID, err)
		}
	}
}
