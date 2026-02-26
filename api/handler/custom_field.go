package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/aegis-imaging/aegis/api/model"
)

// ListCustomFieldDefinitions GET /api/projects/{id}/custom-fields
func (s *Server) ListCustomFieldDefinitions(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	defs, err := model.ListCustomFieldDefinitions(r.Context(), s.db, projectID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	if defs == nil {
		defs = []model.CustomFieldDefinition{}
	}
	s.writeJSON(w, http.StatusOK, defs)
}

// CreateCustomFieldDefinition POST /api/projects/{id}/custom-fields
func (s *Server) CreateCustomFieldDefinition(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")

	var req struct {
		Name      string          `json:"name"`
		FieldType string          `json:"field_type"`
		Options   json.RawMessage `json:"options"`
		Required  bool            `json:"required"`
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
	req.FieldType = strings.TrimSpace(req.FieldType)
	if req.FieldType == "" {
		req.FieldType = "text"
	}
	validTypes := map[string]bool{"text": true, "number": true, "date": true, "select": true}
	if !validTypes[req.FieldType] {
		s.writeError(w, http.StatusBadRequest, "field_type must be text, number, date, or select")
		return
	}
	if req.Options == nil {
		req.Options = json.RawMessage(`[]`)
	}

	d := &model.CustomFieldDefinition{
		ProjectID: projectID,
		Name:      req.Name,
		FieldType: req.FieldType,
		Options:   req.Options,
		Required:  req.Required,
	}
	if err := model.CreateCustomFieldDefinition(r.Context(), s.db, d); err != nil {
		s.writeError(w, http.StatusInternalServerError, "create failed")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "custom_field.created", actorEmail(r), "project", projectID, clientIP(r), map[string]any{
		"name": d.Name, "field_type": d.FieldType,
	})
	s.writeJSON(w, http.StatusCreated, d)
}

// DeleteCustomFieldDefinition DELETE /api/projects/{id}/custom-fields/{fieldID}
func (s *Server) DeleteCustomFieldDefinition(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	fieldID := r.PathValue("fieldID")
	if err := model.DeleteCustomFieldDefinition(r.Context(), s.db, fieldID, projectID); err != nil {
		s.writeError(w, http.StatusInternalServerError, "delete failed")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "custom_field.deleted", actorEmail(r), "project", projectID, clientIP(r), map[string]any{
		"field_id": fieldID,
	})
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ListStudyCustomFields GET /api/studies/{id}/custom-fields
func (s *Server) ListStudyCustomFields(w http.ResponseWriter, r *http.Request) {
	studyID := r.PathValue("id")
	values, err := model.ListStudyCustomFieldValues(r.Context(), s.db, studyID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	if values == nil {
		values = []model.StudyCustomFieldValue{}
	}
	s.writeJSON(w, http.StatusOK, values)
}

// SetStudyCustomField POST /api/studies/{id}/custom-fields
func (s *Server) SetStudyCustomField(w http.ResponseWriter, r *http.Request) {
	studyID := r.PathValue("id")

	var req struct {
		FieldID string `json:"field_id"`
		Value   string `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.FieldID = strings.TrimSpace(req.FieldID)
	if req.FieldID == "" {
		s.writeError(w, http.StatusBadRequest, "field_id is required")
		return
	}

	if err := model.UpsertStudyCustomFieldValue(r.Context(), s.db, studyID, req.FieldID, req.Value); err != nil {
		s.writeError(w, http.StatusInternalServerError, "save failed")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "study.custom_field_set", actorEmail(r), "study", studyID, clientIP(r), map[string]any{
		"field_id": req.FieldID,
	})
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
