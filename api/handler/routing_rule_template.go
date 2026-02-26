package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/aegis-imaging/aegis/api/model"
)

// ListRoutingRuleTemplates GET /api/routing-rule-templates
func (s *Server) ListRoutingRuleTemplates(w http.ResponseWriter, r *http.Request) {
	templates, err := model.ListRoutingRuleTemplates(r.Context(), s.db)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	if templates == nil {
		templates = []model.RoutingRuleTemplate{}
	}
	s.writeJSON(w, http.StatusOK, templates)
}

// CreateRoutingRuleTemplate POST /api/routing-rule-templates
func (s *Server) CreateRoutingRuleTemplate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string  `json:"name"`
		Description string  `json:"description"`
		Action      string  `json:"action"`
		Modality    *string `json:"modality"`
		BodyPart    *string `json:"body_part"`
		Source      *string `json:"source"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		s.writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	req.Action = strings.TrimSpace(req.Action)
	if req.Action == "" {
		s.writeError(w, http.StatusBadRequest, "action is required")
		return
	}

	t := &model.RoutingRuleTemplate{
		Name:        req.Name,
		Description: strings.TrimSpace(req.Description),
		Action:      req.Action,
		Modality:    req.Modality,
		BodyPart:    req.BodyPart,
		Source:      req.Source,
		CreatedBy:   actorEmail(r),
	}
	if err := model.CreateRoutingRuleTemplate(r.Context(), s.db, t); err != nil {
		s.writeError(w, http.StatusInternalServerError, "create failed")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "routing_rule_template.created", actorEmail(r), "routing_rule_template", t.ID, clientIP(r), map[string]any{
		"name": t.Name, "action": t.Action,
	})
	s.writeJSON(w, http.StatusCreated, t)
}

// DeleteRoutingRuleTemplate DELETE /api/routing-rule-templates/{id}
func (s *Server) DeleteRoutingRuleTemplate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := model.DeleteRoutingRuleTemplate(r.Context(), s.db, id); err != nil {
		s.writeError(w, http.StatusInternalServerError, "delete failed")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "routing_rule_template.deleted", actorEmail(r), "routing_rule_template", id, clientIP(r), nil)
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
