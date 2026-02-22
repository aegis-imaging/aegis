// Package webhook delivers HTTP POST callbacks to configured subscribers
// when study events occur. Payloads are signed with HMAC-SHA256 when a
// secret is configured on the subscription.
package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/aegis-imaging/aegis/api/model"
)

// Payload is the JSON body posted to webhook subscriber URLs.
type Payload struct {
	Event           string `json:"event"`
	StudyID         string `json:"study_id"`
	StudyInstanceUID string `json:"study_instance_uid"`
	ProjectID       string `json:"project_id"`
	Timestamp       string `json:"timestamp"`
}

var client = &http.Client{Timeout: 10 * time.Second}

// Deliver fires all enabled webhook subscriptions matching the event and
// project, retrying up to 3 times with exponential backoff. Non-blocking:
// call as a goroutine so delivery never blocks the HTTP handler.
func Deliver(ctx context.Context, db *sql.DB, event string, study *model.Study) {
	subs, err := model.ListEnabledWebhooksForEvent(ctx, db, event, study.ProjectID)
	if err != nil {
		log.Printf("webhook: list subscribers for %s: %v", event, err)
		return
	}
	if len(subs) == 0 {
		return
	}

	payload := Payload{
		Event:           event,
		StudyID:         study.ID,
		StudyInstanceUID: study.StudyInstanceUID,
		ProjectID:       study.ProjectID,
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("webhook: marshal payload: %v", err)
		return
	}

	for _, sub := range subs {
		go deliverOne(sub, body)
	}
}

func deliverOne(sub model.WebhookSubscription, body []byte) {
	delays := []time.Duration{0, 5 * time.Second, 30 * time.Second}
	for attempt, delay := range delays {
		if delay > 0 {
			time.Sleep(delay)
		}
		if err := post(sub, body); err != nil {
			log.Printf("webhook: deliver to %s (attempt %d/3): %v", sub.URL, attempt+1, err)
			continue
		}
		return
	}
	log.Printf("webhook: all 3 attempts failed for subscription %s → %s", sub.ID, sub.URL)
}

func post(sub model.WebhookSubscription, body []byte) error {
	req, err := http.NewRequest("POST", sub.URL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Aegis-Event", "webhook")
	if sub.Secret != "" {
		mac := hmac.New(sha256.New, []byte(sub.Secret))
		mac.Write(body)
		req.Header.Set("X-Aegis-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("non-2xx response: %d", resp.StatusCode)
	}
	return nil
}
