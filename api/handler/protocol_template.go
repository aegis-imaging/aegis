package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/aegis-imaging/aegis/api/model"
)

// ListProtocolTemplates returns all protocol templates for a project.
// GET /api/projects/{projectID}/protocol-templates
func (s *Server) ListProtocolTemplates(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("projectID")
	templates, err := model.ListProtocolTemplatesByProject(r.Context(), s.db, projectID)
	if err != nil {
		log.Printf("list protocol templates: %v", err)
		s.writeError(w, http.StatusInternalServerError, "failed to list templates")
		return
	}
	if templates == nil {
		templates = []model.ProtocolTemplate{}
	}
	s.writeJSON(w, http.StatusOK, templates)
}

// CreateProtocolTemplate creates a new protocol template for a project.
// POST /api/projects/{projectID}/protocol-templates
func (s *Server) CreateProtocolTemplate(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("projectID")
	if _, err := model.GetProjectByID(r.Context(), s.db, projectID); err != nil {
		s.writeError(w, http.StatusNotFound, "project not found")
		return
	}

	var t model.ProtocolTemplate
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if t.Name == "" {
		s.writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	t.ProjectID = projectID
	t.Enabled = true

	if err := model.CreateProtocolTemplate(r.Context(), s.db, &t); err != nil {
		log.Printf("create protocol template: %v", err)
		s.writeError(w, http.StatusInternalServerError, "failed to create template")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "protocol_template.created", actorEmail(r), "protocol_template", t.ID, clientIP(r), map[string]any{
		"name":          t.Name,
		"project_id":    projectID,
		"manufacturer":  t.Manufacturer,
		"model":         t.Model,
		"sequence_type": t.SequenceType,
	})
	s.writeJSON(w, http.StatusCreated, t)
}

// GetProtocolTemplate returns a single protocol template by ID.
// GET /api/protocol-templates/{id}
func (s *Server) GetProtocolTemplate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	t, err := model.GetProtocolTemplateByID(r.Context(), s.db, id)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "template not found")
		return
	}
	s.writeJSON(w, http.StatusOK, t)
}

// UpdateProtocolTemplate updates an existing protocol template.
// PUT /api/protocol-templates/{id}
func (s *Server) UpdateProtocolTemplate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	existing, err := model.GetProtocolTemplateByID(r.Context(), s.db, id)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "template not found")
		return
	}

	var update model.ProtocolTemplate
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	existing.Name = update.Name
	existing.Description = update.Description
	existing.Manufacturer = update.Manufacturer
	existing.Model = update.Model
	existing.SoftwareVersion = update.SoftwareVersion
	existing.SequenceType = update.SequenceType
	existing.Rules = update.Rules
	existing.Enabled = update.Enabled

	if err := model.UpdateProtocolTemplate(r.Context(), s.db, existing); err != nil {
		log.Printf("update protocol template %s: %v", id, err)
		s.writeError(w, http.StatusInternalServerError, "failed to update template")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "protocol_template.updated", actorEmail(r), "protocol_template", id, clientIP(r), nil)
	s.writeJSON(w, http.StatusOK, existing)
}

// DeleteProtocolTemplate deletes a protocol template.
// DELETE /api/protocol-templates/{id}
func (s *Server) DeleteProtocolTemplate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := model.GetProtocolTemplateByID(r.Context(), s.db, id); err != nil {
		s.writeError(w, http.StatusNotFound, "template not found")
		return
	}
	if err := model.DeleteProtocolTemplate(r.Context(), s.db, id); err != nil {
		log.Printf("delete protocol template %s: %v", id, err)
		s.writeError(w, http.StatusInternalServerError, "failed to delete template")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "protocol_template.deleted", actorEmail(r), "protocol_template", id, clientIP(r), nil)
	w.WriteHeader(http.StatusNoContent)
}

// ExportProtocolTemplates returns all protocol templates for a project as a JSON download.
// GET /api/projects/{projectID}/protocol-templates/export
func (s *Server) ExportProtocolTemplates(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("projectID")
	templates, err := model.ListProtocolTemplatesByProject(r.Context(), s.db, projectID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to list templates")
		return
	}
	if templates == nil {
		templates = []model.ProtocolTemplate{}
	}

	payload := map[string]any{
		"project_id": projectID,
		"templates":  templates,
		"count":      len(templates),
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to marshal templates")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", `attachment; filename="protocol-templates.json"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
	model.CreateAuditEntry(r.Context(), s.db, "protocol_template.exported", actorEmail(r), "project", projectID, clientIP(r),
		map[string]any{"count": len(templates)})
}
