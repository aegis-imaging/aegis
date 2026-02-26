package handler

import (
	"encoding/json"
	"net/http"

	"github.com/aegis-imaging/aegis/api/model"
)

type assignStudyRequest struct {
	UserID string `json:"user_id"`
}

// AssignStudy POST /api/studies/{id}/assign
func (s *Server) AssignStudy(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req assignStudyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.UserID == "" {
		s.writeError(w, http.StatusBadRequest, "user_id is required")
		return
	}

	// Verify study exists.
	if _, err := model.GetStudyByID(r.Context(), s.db, id); err != nil {
		s.writeError(w, http.StatusNotFound, "study not found")
		return
	}

	if err := model.AssignStudy(r.Context(), s.db, id, req.UserID); err != nil {
		s.writeError(w, http.StatusInternalServerError, "assignment failed")
		return
	}

	actor := actorEmail(r)
	model.CreateAuditEntry(r.Context(), s.db, "study.assigned", actor, "study", id, clientIP(r), map[string]any{
		"assigned_to": req.UserID,
	})
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// UnassignStudy DELETE /api/studies/{id}/assign
func (s *Server) UnassignStudy(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if _, err := model.GetStudyByID(r.Context(), s.db, id); err != nil {
		s.writeError(w, http.StatusNotFound, "study not found")
		return
	}

	if err := model.UnassignStudy(r.Context(), s.db, id); err != nil {
		s.writeError(w, http.StatusInternalServerError, "unassignment failed")
		return
	}

	actor := actorEmail(r)
	model.CreateAuditEntry(r.Context(), s.db, "study.unassigned", actor, "study", id, clientIP(r), nil)
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
