package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/aegis-imaging/aegis/api/model"
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
