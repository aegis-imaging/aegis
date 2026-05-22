package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/aegis-imaging/aegis/api/model"
)

// ListAuditAnnotations GET /api/audit/{id}/annotations
func (s *Server) ListAuditAnnotations(w http.ResponseWriter, r *http.Request) {
	auditID := r.PathValue("id")
	annotations, err := model.ListAuditAnnotations(r.Context(), s.db, auditID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	if annotations == nil {
		annotations = []model.AuditAnnotation{}
	}
	s.writeJSON(w, http.StatusOK, annotations)
}

// CreateAuditAnnotation POST /api/audit/{id}/annotations
func (s *Server) CreateAuditAnnotation(w http.ResponseWriter, r *http.Request) {
	auditID := r.PathValue("id")

	var req struct {
		Body string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Body = strings.TrimSpace(req.Body)
	if req.Body == "" {
		s.writeError(w, http.StatusBadRequest, "body is required")
		return
	}
	if len(req.Body) > 1000 {
		s.writeError(w, http.StatusBadRequest, "body must be 1000 characters or fewer")
		return
	}

	a := &model.AuditAnnotation{
		AuditID: auditID,
		Author:  actorEmail(r),
		Body:    req.Body,
	}
	if err := model.CreateAuditAnnotation(r.Context(), s.db, a); err != nil {
		s.writeError(w, http.StatusInternalServerError, "create failed")
		return
	}
	s.writeJSON(w, http.StatusCreated, a)
}

// DeleteAuditAnnotation DELETE /api/audit/{id}/annotations/{annotationID}
func (s *Server) DeleteAuditAnnotation(w http.ResponseWriter, r *http.Request) {
	annotationID := r.PathValue("annotationID")
	if err := model.DeleteAuditAnnotation(r.Context(), s.db, annotationID); err != nil {
		s.writeError(w, http.StatusInternalServerError, "delete failed")
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
