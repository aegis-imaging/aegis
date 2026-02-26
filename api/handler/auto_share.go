package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/aegis-imaging/aegis/api/model"
)

type createAutoShareRuleRequest struct {
	RecipientEmail string `json:"recipient_email"`
	ExpiryHours    int    `json:"expiry_hours"`
	Note           string `json:"note"`
}

// ListAutoShareRules GET /api/projects/{projectID}/auto-share-rules
func (s *Server) ListAutoShareRules(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("projectID")
	rules, err := model.ListAutoShareRules(r.Context(), s.db, projectID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	if rules == nil {
		rules = []model.AutoShareRule{}
	}
	s.writeJSON(w, http.StatusOK, rules)
}

// CreateAutoShareRule POST /api/projects/{projectID}/auto-share-rules
func (s *Server) CreateAutoShareRule(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("projectID")

	var req createAutoShareRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	email := strings.TrimSpace(req.RecipientEmail)
	if email == "" {
		s.writeError(w, http.StatusBadRequest, "recipient_email is required")
		return
	}
	if req.ExpiryHours <= 0 {
		req.ExpiryHours = 168
	}

	rule := &model.AutoShareRule{
		ProjectID:      projectID,
		RecipientEmail: email,
		ExpiryHours:    req.ExpiryHours,
		Note:           strings.TrimSpace(req.Note),
		Enabled:        true,
		CreatedBy:      actorEmail(r),
	}
	if err := model.CreateAutoShareRule(r.Context(), s.db, rule); err != nil {
		s.writeError(w, http.StatusInternalServerError, "create failed")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "auto_share_rule.created", actorEmail(r), "auto_share_rule", rule.ID, clientIP(r), map[string]any{
		"project_id": projectID, "recipient_email": email,
	})
	s.writeJSON(w, http.StatusCreated, rule)
}

type updateAutoShareRuleRequest struct {
	Enabled *bool `json:"enabled"`
}

// UpdateAutoShareRule PUT /api/auto-share-rules/{id}
func (s *Server) UpdateAutoShareRule(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req updateAutoShareRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Enabled == nil {
		s.writeError(w, http.StatusBadRequest, "enabled is required")
		return
	}
	if err := model.UpdateAutoShareRule(r.Context(), s.db, id, *req.Enabled); err != nil {
		s.writeError(w, http.StatusInternalServerError, "update failed")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "auto_share_rule.updated", actorEmail(r), "auto_share_rule", id, clientIP(r), map[string]any{
		"enabled": *req.Enabled,
	})
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// DeleteAutoShareRule DELETE /api/auto-share-rules/{id}
func (s *Server) DeleteAutoShareRule(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := model.DeleteAutoShareRule(r.Context(), s.db, id); err != nil {
		s.writeError(w, http.StatusInternalServerError, "delete failed")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "auto_share_rule.deleted", actorEmail(r), "auto_share_rule", id, clientIP(r), nil)
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
