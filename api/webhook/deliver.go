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
	Event            string            `json:"event"`
	StudyID          string            `json:"study_id"`
	StudyInstanceUID string            `json:"study_instance_uid"`
	ProjectID        string            `json:"project_id"`
	Timestamp        string            `json:"timestamp"`
	FHIRImagingStudy *FHIRImagingStudy `json:"fhir_imaging_study,omitempty"`
}

var client = &http.Client{Timeout: 10 * time.Second}

// Deliver fires all enabled webhook subscriptions matching the event and
// project, retrying up to 3 times with exponential backoff. Non-blocking:
// call as a goroutine so delivery never blocks the HTTP handler.
//
// Each subscription chooses its payload shape via payload_format:
//   - "aegis" (default) — full AEGIS envelope with embedded fhir_imaging_study
//   - "fhir"            — bare FHIR R4 ImagingStudy resource as the body
func Deliver(ctx context.Context, db *sql.DB, event string, study *model.Study) {
	subs, err := model.ListEnabledWebhooksForEvent(ctx, db, event, study.ProjectID)
	if err != nil {
		log.Printf("webhook: list subscribers for %s: %v", event, err)
		return
	}
	if len(subs) == 0 {
		return
	}

	fhir := buildFHIRImagingStudy(study)
	aegisPayload := Payload{
		Event:            event,
		StudyID:          study.ID,
		StudyInstanceUID: study.StudyInstanceUID,
		ProjectID:        study.ProjectID,
		Timestamp:        time.Now().UTC().Format(time.RFC3339),
		FHIRImagingStudy: &fhir,
	}
	aegisBody, err := json.Marshal(aegisPayload)
	if err != nil {
		log.Printf("webhook: marshal AEGIS payload: %v", err)
		return
	}
	fhirBody, err := json.Marshal(fhir)
	if err != nil {
		log.Printf("webhook: marshal FHIR payload: %v", err)
		return
	}

	for _, sub := range subs {
		body := aegisBody
		if model.NormalizePayloadFormat(sub.PayloadFormat) == model.PayloadFormatFHIR {
			body = fhirBody
		}
		go deliverOne(db, sub, event, body)
	}
}

func deliverOne(db *sql.DB, sub model.WebhookSubscription, event string, body []byte) {
	delays := []time.Duration{0, 5 * time.Second, 30 * time.Second}
	for attempt, delay := range delays {
		if delay > 0 {
			time.Sleep(delay)
		}
		statusCode, err := post(sub, body)
		rec := &model.WebhookDelivery{
			SubscriptionID: sub.ID,
			Event:          event,
			URL:            sub.URL,
			Attempt:        attempt + 1,
			Success:        err == nil,
		}
		if statusCode != 0 {
			rec.StatusCode = &statusCode
		}
		if err != nil {
			msg := err.Error()
			rec.ErrorMessage = &msg
		}
		model.RecordWebhookDelivery(context.Background(), db, rec)

		if err != nil {
			log.Printf("webhook: deliver to %s (attempt %d/3): %v", sub.URL, attempt+1, err)
			continue
		}
		return
	}
	log.Printf("webhook: all 3 attempts failed for subscription %s → %s", sub.ID, sub.URL)
}

// PostTest sends a single synchronous test delivery and returns the HTTP status
// code and any error. The caller is responsible for recording the delivery log entry.
//
// The body sent reflects the subscription's payload_format: AEGIS envelope
// for "aegis" (default), or the embedded FHIR ImagingStudy resource for
// "fhir". This way a "Test" button in the dashboard exercises the same
// shape a real event would use.
func PostTest(sub model.WebhookSubscription, payload Payload) (int, error) {
	var body []byte
	var err error
	if model.NormalizePayloadFormat(sub.PayloadFormat) == model.PayloadFormatFHIR {
		if payload.FHIRImagingStudy == nil {
			return 0, fmt.Errorf("fhir payload_format requires a fhir_imaging_study")
		}
		body, err = json.Marshal(payload.FHIRImagingStudy)
	} else {
		body, err = json.Marshal(payload)
	}
	if err != nil {
		return 0, fmt.Errorf("marshal payload: %w", err)
	}
	return post(sub, body)
}

// Add the helper after PostTest so it sits near `post`. Keeping it package-local
// so callers don't depend on internal Content-Type plumbing.
func contentTypeFor(format string) string {
	if model.NormalizePayloadFormat(format) == model.PayloadFormatFHIR {
		return "application/fhir+json"
	}
	return "application/json"
}

func post(sub model.WebhookSubscription, body []byte) (int, error) {
	req, err := http.NewRequest("POST", sub.URL, bytes.NewReader(body))
	if err != nil {
		return 0, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", contentTypeFor(sub.PayloadFormat))
	req.Header.Set("X-Aegis-Event", "webhook")
	if sub.Secret != "" {
		mac := hmac.New(sha256.New, []byte(sub.Secret))
		mac.Write(body)
		req.Header.Set("X-Aegis-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp.StatusCode, fmt.Errorf("non-2xx response: %d", resp.StatusCode)
	}
	return resp.StatusCode, nil
}
