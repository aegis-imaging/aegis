// Package digest provides a background scheduler that sends periodic email
// summaries to configured subscribers.
package digest

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/aegis-imaging/aegis/api/email"
	"github.com/aegis-imaging/aegis/api/model"
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
	nowUTC := time.Now().UTC()

	subs, err := model.ListEnabledDigestSubscriptions(ctx, db)
	if err != nil {
		log.Printf("digest: list enabled subscriptions: %v", err)
		return
	}
	if len(subs) == 0 {
		return
	}

	for _, sub := range subs {
		if !isDigestDue(sub.Frequency, sub.LastSentAt, nowUTC) {
			continue
		}
		if err := sendOne(ctx, db, mailer, sub, nowUTC); err != nil {
			log.Printf("digest: send to %s (project %s): %v", sub.Email, sub.ProjectName, err)
			// Continue processing remaining subscriptions even if one fails.
			continue
		}
		if err := model.UpdateDigestLastSent(ctx, db, sub.ID); err != nil {
			log.Printf("digest: update last_sent_at for %s: %v", sub.ID, err)
		}
	}
}

func sendOne(ctx context.Context, db *sql.DB, mailer *email.Client, sub model.DigestSubscription, nowUTC time.Time) error {
	sinceUTC, _, period := periodRange(sub.Frequency, nowUTC)
	stats, err := model.GetDigestStats(ctx, db, sub.ProjectID, sinceUTC)
	if err != nil {
		return fmt.Errorf("query stats: %w", err)
	}

	subject, body := email.DigestSummary(
		sub.ProjectName, sub.Frequency, period,
		stats.Received, stats.Approved, stats.Rejected, stats.Pending, stats.SharesCreated,
	)

	if err := mailer.Send(ctx, sub.Email, subject, body); err != nil {
		return fmt.Errorf("send email: %w", err)
	}
	return nil
}

func isDigestDue(frequency string, lastSentAt *time.Time, nowUTC time.Time) bool {
	if lastSentAt == nil {
		return true
	}
	nowUTC = nowUTC.UTC()
	lastUTC := lastSentAt.UTC()
	switch frequency {
	case "monthly":
		return lastUTC.Before(monthlyCutoffUTC(nowUTC))
	default: // weekly
		return lastUTC.Before(nowUTC.AddDate(0, 0, -7))
	}
}

// monthlyCutoffUTC returns the same day/time in the previous month, clamping to
// the last day when the previous month is shorter (e.g., Mar 31 -> Feb 28/29).
func monthlyCutoffUTC(nowUTC time.Time) time.Time {
	nowUTC = nowUTC.UTC()
	year, month, day := nowUTC.Date()
	hour, minute, second := nowUTC.Clock()
	nsec := nowUTC.Nanosecond()

	prevMonthStart := time.Date(year, month, 1, hour, minute, second, nsec, time.UTC).AddDate(0, -1, 0)
	thisMonthStart := time.Date(year, month, 1, hour, minute, second, nsec, time.UTC)
	lastDayPrevMonth := thisMonthStart.AddDate(0, 0, -1).Day()
	if day > lastDayPrevMonth {
		day = lastDayPrevMonth
	}

	return time.Date(prevMonthStart.Year(), prevMonthStart.Month(), day, hour, minute, second, nsec, time.UTC)
}

// periodRange returns [since, until] for the digest frequency, plus a UTC label.
// The label always includes explicit UTC timestamps to avoid timezone ambiguity.
func periodRange(frequency string, nowUTC time.Time) (sinceUTC, untilUTC time.Time, label string) {
	nowUTC = nowUTC.UTC()
	untilUTC = nowUTC

	switch frequency {
	case "monthly":
		sinceUTC = monthlyCutoffUTC(nowUTC)
	default: // weekly
		sinceUTC = nowUTC.AddDate(0, 0, -7)
	}
	label = sinceUTC.Format("2006-01-02 15:04 UTC") + " – " + untilUTC.Format("2006-01-02 15:04 UTC")
	return sinceUTC, untilUTC, label
}
