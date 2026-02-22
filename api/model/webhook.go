package model

import (
	"context"
	"database/sql"
	"encoding/json"
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
