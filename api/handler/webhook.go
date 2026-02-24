package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/webhook"
)

// ValidWebhookEvents is the set of events subscribers can listen to.
var ValidWebhookEvents = map[string]bool{
	"study.approved":      true,
	"study.rejected":      true,
	"study.phi_flagged":   true,
	"study.export_complete": true,
	"study.stuck":         true,
}

type webhookRequest struct {
	ProjectID *string  `json:"project_id"`
	URL       string   `json:"url"`
	Events    []string `json:"events"`
	Secret    string   `json:"secret"`
	Enabled   *bool    `json:"enabled"`
}

// ListWebhooks GET /api/webhook-subscriptions
func (s *Server) ListWebhooks(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project_id")
	subs, err := model.ListWebhookSubscriptions(r.Context(), s.db, projectID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	if subs == nil {
		subs = []model.WebhookSubscription{}
	}
	s.writeJSON(w, http.StatusOK, subs)
}

// CreateWebhook POST /api/webhook-subscriptions
func (s *Server) CreateWebhook(w http.ResponseWriter, r *http.Request) {
	var req webhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validateWebhookRequest(req); err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	sub := &model.WebhookSubscription{
		ProjectID: req.ProjectID,
		URL:       strings.TrimSpace(req.URL),
		Events:    req.Events,
		Secret:    req.Secret,
		Enabled:   enabled,
	}
	if err := model.CreateWebhookSubscription(r.Context(), s.db, sub); err != nil {
		s.writeError(w, http.StatusInternalServerError, "create failed")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "webhook.created", actorEmail(r), "webhook", sub.ID, clientIP(r), map[string]any{
		"url": sub.URL, "events": sub.Events,
	})
	s.writeJSON(w, http.StatusCreated, sub)
}

// GetWebhook GET /api/webhook-subscriptions/{id}
func (s *Server) GetWebhook(w http.ResponseWriter, r *http.Request) {
	sub, err := model.GetWebhookSubscription(r.Context(), s.db, r.PathValue("id"))
	if err != nil {
		s.writeError(w, http.StatusNotFound, "not found")
		return
	}
	s.writeJSON(w, http.StatusOK, sub)
}

// UpdateWebhook PUT /api/webhook-subscriptions/{id}
func (s *Server) UpdateWebhook(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	sub, err := model.GetWebhookSubscription(r.Context(), s.db, id)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "not found")
		return
	}
	var req webhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validateWebhookRequest(req); err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	sub.URL = strings.TrimSpace(req.URL)
	sub.Events = req.Events
	sub.Secret = req.Secret
	if req.Enabled != nil {
		sub.Enabled = *req.Enabled
	}
	if err := model.UpdateWebhookSubscription(r.Context(), s.db, sub); err != nil {
		s.writeError(w, http.StatusInternalServerError, "update failed")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "webhook.updated", actorEmail(r), "webhook", id, clientIP(r), nil)
	s.writeJSON(w, http.StatusOK, sub)
}

// GetWebhookDeliveries GET /api/webhook-subscriptions/{id}/deliveries
// Returns the most recent delivery log entries for a webhook subscription.
func (s *Server) GetWebhookDeliveries(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := model.GetWebhookSubscription(r.Context(), s.db, id); err != nil {
		s.writeError(w, http.StatusNotFound, "not found")
		return
	}
	deliveries, err := model.ListWebhookDeliveries(r.Context(), s.db, id, 0)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	if deliveries == nil {
		deliveries = []model.WebhookDelivery{}
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"deliveries": deliveries})
}

// DeleteWebhook DELETE /api/webhook-subscriptions/{id}
func (s *Server) DeleteWebhook(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := model.GetWebhookSubscription(r.Context(), s.db, id); err != nil {
		s.writeError(w, http.StatusNotFound, "not found")
		return
	}
	if err := model.DeleteWebhookSubscription(r.Context(), s.db, id); err != nil {
		s.writeError(w, http.StatusInternalServerError, "delete failed")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "webhook.deleted", actorEmail(r), "webhook", id, clientIP(r), nil)
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// TestWebhookDelivery POST /api/webhook-subscriptions/{id}/test
// Sends a synthetic test payload to the subscriber's URL and returns the delivery outcome.
func (s *Server) TestWebhookDelivery(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	sub, err := model.GetWebhookSubscription(r.Context(), s.db, id)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "not found")
		return
	}

	testEvent := "study.approved"
	payload := webhook.Payload{
		Event:            testEvent,
		StudyID:          "00000000-0000-0000-0000-000000000000",
		StudyInstanceUID: "1.2.3.4.5.6.7.8.9.test",
		ProjectID:        "",
		Timestamp:        time.Now().UTC().Format(time.RFC3339),
	}
	if sub.ProjectID != nil {
		payload.ProjectID = *sub.ProjectID
	}

	statusCode, deliveryErr := webhook.PostTest(*sub, payload)

	success := deliveryErr == nil
	rec := &model.WebhookDelivery{
		SubscriptionID: sub.ID,
		Event:          testEvent,
		URL:            sub.URL,
		Attempt:        1,
		Success:        success,
	}
	if statusCode != 0 {
		rec.StatusCode = &statusCode
	}
	if deliveryErr != nil {
		msg := deliveryErr.Error()
		rec.ErrorMessage = &msg
	}
	model.RecordWebhookDelivery(r.Context(), s.db, rec)
	model.CreateAuditEntry(r.Context(), s.db, "webhook.test", actorEmail(r), "webhook", id, clientIP(r), map[string]any{
		"url": sub.URL, "success": success, "status_code": statusCode,
	})

	resp := map[string]any{
		"success":     success,
		"status_code": statusCode,
		"url":         sub.URL,
	}
	if deliveryErr != nil {
		resp["error"] = deliveryErr.Error()
	}
	s.writeJSON(w, http.StatusOK, resp)
}

// GetWebhookStats GET /api/webhook-subscriptions/{id}/stats
func (s *Server) GetWebhookStats(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := model.GetWebhookSubscription(r.Context(), s.db, id); err != nil {
		s.writeError(w, http.StatusNotFound, "not found")
		return
	}
	stats, err := model.GetWebhookStats(r.Context(), s.db, id)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "stats query failed")
		return
	}
	s.writeJSON(w, http.StatusOK, stats)
}

// ListAllDeliveries GET /api/webhook-deliveries
func (s *Server) ListAllDeliveries(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	subID := q.Get("subscription_id")

	var successOnly *bool
	if v := q.Get("success"); v == "true" {
		t := true
		successOnly = &t
	} else if v == "false" {
		f := false
		successOnly = &f
	}

	limit, offset := 50, 0
	if v := q.Get("limit"); v != "" {
		if n, err := parseIntParam(v); err == nil && n > 0 {
			limit = n
		}
	}
	if v := q.Get("offset"); v != "" {
		if n, err := parseIntParam(v); err == nil && n >= 0 {
			offset = n
		}
	}

	deliveries, total, err := model.ListAllWebhookDeliveries(r.Context(), s.db, subID, successOnly, limit, offset)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	if deliveries == nil {
		deliveries = []model.WebhookDelivery{}
	}
	s.writeJSON(w, http.StatusOK, map[string]any{
		"deliveries": deliveries,
		"total":      total,
		"limit":      limit,
		"offset":     offset,
	})
}

// RetryDelivery POST /api/webhook-deliveries/{id}/retry
// Re-sends the same event to the subscription's current URL with a fresh timestamp.
func (s *Server) RetryDelivery(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	delivery, err := model.GetWebhookDelivery(r.Context(), s.db, id)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "delivery not found")
		return
	}

	sub, err := model.GetWebhookSubscription(r.Context(), s.db, delivery.SubscriptionID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "subscription not found")
		return
	}

	payload := webhook.Payload{
		Event:     delivery.Event,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	statusCode, deliveryErr := webhook.PostTest(*sub, payload)

	success := deliveryErr == nil
	rec := &model.WebhookDelivery{
		SubscriptionID: sub.ID,
		Event:          delivery.Event,
		URL:            sub.URL,
		Attempt:        1,
		Success:        success,
	}
	if statusCode != 0 {
		rec.StatusCode = &statusCode
	}
	if deliveryErr != nil {
		msg := deliveryErr.Error()
		rec.ErrorMessage = &msg
	}
	model.RecordWebhookDelivery(r.Context(), s.db, rec)
	model.CreateAuditEntry(r.Context(), s.db, "webhook.delivery_retried", actorEmail(r), "webhook", sub.ID, clientIP(r), map[string]any{
		"original_delivery_id": id, "success": success,
	})

	resp := map[string]any{"success": success, "status_code": statusCode}
	if deliveryErr != nil {
		resp["error"] = deliveryErr.Error()
	}
	s.writeJSON(w, http.StatusOK, resp)
}

func parseIntParam(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}

func validateWebhookRequest(req webhookRequest) error {
	if strings.TrimSpace(req.URL) == "" {
		return errMsg("url is required")
	}
	if !strings.HasPrefix(req.URL, "http://") && !strings.HasPrefix(req.URL, "https://") {
		return errMsg("url must start with http:// or https://")
	}
	if len(req.Events) == 0 {
		return errMsg("events must not be empty")
	}
	for _, e := range req.Events {
		if !ValidWebhookEvents[e] {
			return errMsg("unknown event: " + e + "; valid events: study.approved, study.rejected, study.phi_flagged, study.export_complete, study.stuck")
		}
	}
	return nil
}

type errMsg string

func (e errMsg) Error() string { return string(e) }
