package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/aegis-imaging/aegis/api/model"
)

// ListExportShareTemplates GET /api/export-share-templates
func (s *Server) ListExportShareTemplates(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project_id")
	templates, err := model.ListExportShareTemplates(r.Context(), s.db, projectID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	if templates == nil {
		templates = []model.ExportShareTemplate{}
	}
	s.writeJSON(w, http.StatusOK, templates)
}

// CreateExportShareTemplate POST /api/export-share-templates
func (s *Server) CreateExportShareTemplate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ProjectID      *string `json:"project_id"`
		Name           string  `json:"name"`
		RecipientEmail string  `json:"recipient_email"`
		Note           string  `json:"note"`
		ExpiryHours    int     `json:"expiry_hours"`
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
	if len(req.Name) > 100 {
		s.writeError(w, http.StatusBadRequest, "name must be 100 characters or fewer")
		return
	}
	if req.ExpiryHours <= 0 {
		req.ExpiryHours = 72
	}

	t := &model.ExportShareTemplate{
		ProjectID:      req.ProjectID,
		Name:           req.Name,
		RecipientEmail: strings.TrimSpace(req.RecipientEmail),
		Note:           strings.TrimSpace(req.Note),
		ExpiryHours:    req.ExpiryHours,
		CreatedBy:      actorEmail(r),
	}
	if err := model.CreateExportShareTemplate(r.Context(), s.db, t); err != nil {
		s.writeError(w, http.StatusInternalServerError, "create failed")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "export_share_template.created", actorEmail(r), "export_share_template", t.ID, clientIP(r), map[string]any{
		"name": t.Name,
	})
	s.writeJSON(w, http.StatusCreated, t)
}

// DeleteExportShareTemplate DELETE /api/export-share-templates/{id}
func (s *Server) DeleteExportShareTemplate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := model.DeleteExportShareTemplate(r.Context(), s.db, id); err != nil {
		s.writeError(w, http.StatusInternalServerError, "delete failed")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "export_share_template.deleted", actorEmail(r), "export_share_template", id, clientIP(r), nil)
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
