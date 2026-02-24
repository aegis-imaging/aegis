package model

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// WebhookSubscription is a configured HTTP endpoint that receives POST callbacks
// when study events occur. Events are delivered with an HMAC-SHA256 signature
// header so receivers can verify the payload origin.
type WebhookSubscription struct {
	ID        string    `json:"id"`
	ProjectID *string   `json:"project_id,omitempty"`
	URL       string    `json:"url"`
	Events    []string  `json:"events"`
	Secret    string    `json:"secret,omitempty"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func scanWebhook(row scannable, w *WebhookSubscription) error {
	var eventsJSON []byte
	if err := row.Scan(
		&w.ID, &w.ProjectID, &w.URL, &eventsJSON, &w.Secret, &w.Enabled,
		&w.CreatedAt, &w.UpdatedAt,
	); err != nil {
		return err
	}
	return json.Unmarshal(eventsJSON, &w.Events)
}

func eventsJSON(events []string) ([]byte, error) {
	if events == nil {
		events = []string{}
	}
	return json.Marshal(events)
}

const webhookColumns = `id, project_id, url, events, secret, enabled, created_at, updated_at`

func CreateWebhookSubscription(ctx context.Context, db *sql.DB, w *WebhookSubscription) error {
	ej, err := eventsJSON(w.Events)
	if err != nil {
		return err
	}
	return db.QueryRowContext(ctx, `
		INSERT INTO webhook_subscriptions (project_id, url, events, secret, enabled)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at`,
		w.ProjectID, w.URL, ej, w.Secret, w.Enabled,
	).Scan(&w.ID, &w.CreatedAt, &w.UpdatedAt)
}

func ListWebhookSubscriptions(ctx context.Context, db *sql.DB, projectID string) ([]WebhookSubscription, error) {
	var rows *sql.Rows
	var err error
	if projectID != "" {
		rows, err = db.QueryContext(ctx, `SELECT `+webhookColumns+` FROM webhook_subscriptions WHERE project_id = $1 ORDER BY created_at DESC`, projectID)
	} else {
		rows, err = db.QueryContext(ctx, `SELECT `+webhookColumns+` FROM webhook_subscriptions ORDER BY created_at DESC`)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var subs []WebhookSubscription
	for rows.Next() {
		var w WebhookSubscription
		if err := scanWebhook(rows, &w); err != nil {
			return nil, err
		}
		subs = append(subs, w)
	}
	return subs, rows.Err()
}

func GetWebhookSubscription(ctx context.Context, db *sql.DB, id string) (*WebhookSubscription, error) {
	var w WebhookSubscription
	if err := scanWebhook(db.QueryRowContext(ctx, `SELECT `+webhookColumns+` FROM webhook_subscriptions WHERE id = $1`, id), &w); err != nil {
		return nil, err
	}
	return &w, nil
}

func UpdateWebhookSubscription(ctx context.Context, db *sql.DB, w *WebhookSubscription) error {
	ej, err := eventsJSON(w.Events)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, `
		UPDATE webhook_subscriptions
		SET url=$1, events=$2, secret=$3, enabled=$4, updated_at=now()
		WHERE id=$5`,
		w.URL, ej, w.Secret, w.Enabled, w.ID)
	return err
}

func DeleteWebhookSubscription(ctx context.Context, db *sql.DB, id string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM webhook_subscriptions WHERE id = $1`, id)
	return err
}

// WebhookDelivery is one recorded delivery attempt for a webhook subscription.
type WebhookDelivery struct {
	ID             string     `json:"id"`
	SubscriptionID string     `json:"subscription_id"`
	Event          string     `json:"event"`
	URL            string     `json:"url"`
	Attempt        int        `json:"attempt"`
	StatusCode     *int       `json:"status_code,omitempty"`
	Success        bool       `json:"success"`
	ErrorMessage   *string    `json:"error_message,omitempty"`
	DeliveredAt    time.Time  `json:"delivered_at"`
}

// RecordWebhookDelivery persists one delivery attempt. Non-fatal on error (best-effort log).
func RecordWebhookDelivery(ctx context.Context, db *sql.DB, d *WebhookDelivery) {
	_, err := db.ExecContext(ctx, `
		INSERT INTO webhook_deliveries (subscription_id, event, url, attempt, status_code, success, error_message)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		d.SubscriptionID, d.Event, d.URL, d.Attempt, d.StatusCode, d.Success, d.ErrorMessage)
	if err != nil {
		// Delivery log is best-effort; don't propagate.
		_ = err
	}
}

// ListWebhookDeliveries returns the most recent delivery attempts for a subscription.
func ListWebhookDeliveries(ctx context.Context, db *sql.DB, subscriptionID string, limit int) ([]WebhookDelivery, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := db.QueryContext(ctx, `
		SELECT id, subscription_id, event, url, attempt, status_code, success, error_message, delivered_at
		FROM webhook_deliveries
		WHERE subscription_id = $1
		ORDER BY delivered_at DESC
		LIMIT $2`, subscriptionID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var deliveries []WebhookDelivery
	for rows.Next() {
		var d WebhookDelivery
		if err := rows.Scan(&d.ID, &d.SubscriptionID, &d.Event, &d.URL, &d.Attempt,
			&d.StatusCode, &d.Success, &d.ErrorMessage, &d.DeliveredAt); err != nil {
			return nil, err
		}
		deliveries = append(deliveries, d)
	}
	return deliveries, rows.Err()
}

// GetWebhookDelivery returns a single delivery record by ID.
func GetWebhookDelivery(ctx context.Context, db *sql.DB, id string) (*WebhookDelivery, error) {
	var d WebhookDelivery
	err := db.QueryRowContext(ctx, `
		SELECT id, subscription_id, event, url, attempt, status_code, success, error_message, delivered_at
		FROM webhook_deliveries WHERE id = $1`, id).
		Scan(&d.ID, &d.SubscriptionID, &d.Event, &d.URL, &d.Attempt,
			&d.StatusCode, &d.Success, &d.ErrorMessage, &d.DeliveredAt)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// WebhookStats aggregates delivery metrics for a single webhook subscription.
type WebhookStats struct {
	SubscriptionID    string         `json:"subscription_id"`
	TotalDeliveries   int            `json:"total_deliveries"`
	Successful        int            `json:"successful"`
	Failed            int            `json:"failed"`
	SuccessRatePct    float64        `json:"success_rate_pct"`
	LastDeliveryAt    *time.Time     `json:"last_delivery_at,omitempty"`
	DeliveriesByEvent map[string]int `json:"deliveries_by_event"`
}

// GetWebhookStats computes aggregate delivery metrics for a subscription.
func GetWebhookStats(ctx context.Context, db *sql.DB, subscriptionID string) (*WebhookStats, error) {
	var total, successful int
	var lastAt *time.Time
	err := db.QueryRowContext(ctx, `
		SELECT COUNT(*),
		       COUNT(*) FILTER (WHERE success = true),
		       MAX(delivered_at)
		FROM webhook_deliveries
		WHERE subscription_id = $1`, subscriptionID).
		Scan(&total, &successful, &lastAt)
	if err != nil {
		return nil, err
	}

	eventRows, err := db.QueryContext(ctx, `
		SELECT event, COUNT(*) FROM webhook_deliveries
		WHERE subscription_id = $1
		GROUP BY event`, subscriptionID)
	if err != nil {
		return nil, err
	}
	defer eventRows.Close()
	byEvent := make(map[string]int)
	for eventRows.Next() {
		var ev string
		var cnt int
		if err := eventRows.Scan(&ev, &cnt); err != nil {
			return nil, err
		}
		byEvent[ev] = cnt
	}
	if err := eventRows.Err(); err != nil {
		return nil, err
	}

	failed := total - successful
	var rate float64
	if total > 0 {
		rate = float64(successful) / float64(total) * 100
	}
	return &WebhookStats{
		SubscriptionID:    subscriptionID,
		TotalDeliveries:   total,
		Successful:        successful,
		Failed:            failed,
		SuccessRatePct:    rate,
		LastDeliveryAt:    lastAt,
		DeliveriesByEvent: byEvent,
	}, nil
}

// ListAllWebhookDeliveries returns delivery records across all subscriptions with optional filters.
func ListAllWebhookDeliveries(ctx context.Context, db *sql.DB, subscriptionID string, successOnly *bool, limit, offset int) ([]WebhookDelivery, int, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	where := "1=1"
	args := []any{}
	argIdx := 1

	if subscriptionID != "" {
		where += fmt.Sprintf(" AND subscription_id = $%d", argIdx)
		args = append(args, subscriptionID)
		argIdx++
	}
	if successOnly != nil {
		where += fmt.Sprintf(" AND success = $%d", argIdx)
		args = append(args, *successOnly)
		argIdx++
	}

	var total int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM webhook_deliveries WHERE "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, limit, offset)
	rows, err := db.QueryContext(ctx, `
		SELECT id, subscription_id, event, url, attempt, status_code, success, error_message, delivered_at
		FROM webhook_deliveries
		WHERE `+where+`
		ORDER BY delivered_at DESC
		LIMIT $`+fmt.Sprintf("%d", argIdx)+` OFFSET $`+fmt.Sprintf("%d", argIdx+1), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var deliveries []WebhookDelivery
	for rows.Next() {
		var d WebhookDelivery
		if err := rows.Scan(&d.ID, &d.SubscriptionID, &d.Event, &d.URL, &d.Attempt,
			&d.StatusCode, &d.Success, &d.ErrorMessage, &d.DeliveredAt); err != nil {
			return nil, 0, err
		}
		deliveries = append(deliveries, d)
	}
	return deliveries, total, rows.Err()
}

// ListEnabledWebhooksForEvent returns all enabled webhook subscriptions that
// subscribe to the given event, optionally scoped to a project.
func ListEnabledWebhooksForEvent(ctx context.Context, db *sql.DB, event, projectID string) ([]WebhookSubscription, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT `+webhookColumns+`
		FROM webhook_subscriptions
		WHERE enabled = true
		  AND events @> $1::jsonb
		  AND (project_id IS NULL OR project_id = $2)`,
		`"`+event+`"`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var subs []WebhookSubscription
	for rows.Next() {
		var w WebhookSubscription
		if err := scanWebhook(rows, &w); err != nil {
			return nil, err
		}
		subs = append(subs, w)
	}
	return subs, rows.Err()
}
