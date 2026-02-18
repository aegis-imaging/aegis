// Package digest provides a background scheduler that sends periodic email
// summaries to configured subscribers.
package digest

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/msenjem/aegis/api/email"
	"github.com/msenjem/aegis/api/model"
)

// Start launches a background goroutine that checks for due digest subscriptions
// once per hour and sends summary emails. The goroutine stops when ctx is cancelled.
func Start(ctx context.Context, db *sql.DB, mailer *email.Client) {
	go run(ctx, db, mailer)
}

func run(ctx context.Context, db *sql.DB, mailer *email.Client) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	// Run once immediately on startup so a freshly seeded DB gets digests quickly
	// in development. In production the first tick will fire within an hour.
	send(ctx, db, mailer)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			send(ctx, db, mailer)
		}
	}
}

func send(ctx context.Context, db *sql.DB, mailer *email.Client) {
	subs, err := model.ListDueSubscriptions(ctx, db)
	if err != nil {
		log.Printf("digest: list due subscriptions: %v", err)
		return
	}
	if len(subs) == 0 {
		return
	}

	for _, sub := range subs {
		if err := sendOne(ctx, db, mailer, sub); err != nil {
			log.Printf("digest: send to %s (project %s): %v", sub.Email, sub.ProjectName, err)
			// Continue processing remaining subscriptions even if one fails.
			continue
		}
		if err := model.UpdateDigestLastSent(ctx, db, sub.ID); err != nil {
			log.Printf("digest: update last_sent_at for %s: %v", sub.ID, err)
		}
	}
}

func sendOne(ctx context.Context, db *sql.DB, mailer *email.Client, sub model.DigestSubscription) error {
	since := sinceTime(sub.Frequency)
	stats, err := model.GetDigestStats(ctx, db, sub.ProjectID, since)
	if err != nil {
		return fmt.Errorf("query stats: %w", err)
	}

	periodLabel := periodLabel(sub.Frequency, since)
	subject, body := email.DigestSummary(
		sub.ProjectName, sub.Frequency, periodLabel,
		stats.Received, stats.Approved, stats.Rejected, stats.Pending, stats.SharesCreated,
	)

	if err := mailer.Send(ctx, sub.Email, subject, body); err != nil {
		return fmt.Errorf("send email: %w", err)
	}
	return nil
}

// sinceTime returns the start of the period for the given frequency.
func sinceTime(frequency string) time.Time {
	now := time.Now().UTC()
	switch frequency {
	case "monthly":
		return now.AddDate(0, -1, 0)
	default: // weekly
		return now.AddDate(0, 0, -7)
	}
}

// periodLabel returns a human-readable label for the digest period.
func periodLabel(frequency string, since time.Time) string {
	now := time.Now().UTC()
	switch frequency {
	case "monthly":
		return since.Format("Jan 2006") + " – " + now.Format("Jan 2006")
	default:
		return since.Format("2006-01-02") + " – " + now.Format("2006-01-02")
	}
}
