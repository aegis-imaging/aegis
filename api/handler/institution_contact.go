package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/aegis-imaging/aegis/api/model"
)

// ListInstitutionContacts GET /api/institutions/{id}/contacts
func (s *Server) ListInstitutionContacts(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	contacts, err := model.ListInstitutionContacts(r.Context(), s.db, id)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	if contacts == nil {
		contacts = []model.InstitutionContact{}
	}
	s.writeJSON(w, http.StatusOK, contacts)
}

// CreateInstitutionContact POST /api/institutions/{id}/contacts
func (s *Server) CreateInstitutionContact(w http.ResponseWriter, r *http.Request) {
	instID := r.PathValue("id")

	var req struct {
		Name  string `json:"name"`
		Email string `json:"email"`
		Phone string `json:"phone"`
		Role  string `json:"role"`
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
	if len(req.Name) > 200 {
		s.writeError(w, http.StatusBadRequest, "name must be 200 characters or fewer")
		return
	}

	c := &model.InstitutionContact{
		InstitutionID: instID,
		Name:          req.Name,
		Email:         strings.TrimSpace(req.Email),
		Phone:         strings.TrimSpace(req.Phone),
		Role:          strings.TrimSpace(req.Role),
	}
	if err := model.CreateInstitutionContact(r.Context(), s.db, c); err != nil {
		s.writeError(w, http.StatusInternalServerError, "create failed")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "institution.contact_added", actorEmail(r), "institution", instID, clientIP(r), map[string]any{
		"contact_name": c.Name,
	})
	s.writeJSON(w, http.StatusCreated, c)
}

// DeleteInstitutionContact DELETE /api/institutions/{id}/contacts/{contactID}
func (s *Server) DeleteInstitutionContact(w http.ResponseWriter, r *http.Request) {
	instID := r.PathValue("id")
	contactID := r.PathValue("contactID")
	if err := model.DeleteInstitutionContact(r.Context(), s.db, contactID, instID); err != nil {
		s.writeError(w, http.StatusInternalServerError, "delete failed")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "institution.contact_removed", actorEmail(r), "institution", instID, clientIP(r), map[string]any{
		"contact_id": contactID,
	})
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
