package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/aegis-imaging/aegis/api/model"
)

type addLabelRequest struct {
	Label string `json:"label"`
}

// ListStudyLabels GET /api/studies/{id}/labels
func (s *Server) ListStudyLabels(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	labels, err := model.ListStudyLabels(r.Context(), s.db, id)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	if labels == nil {
		labels = []model.StudyLabel{}
	}
	s.writeJSON(w, http.StatusOK, labels)
}

// AddStudyLabel POST /api/studies/{id}/labels
func (s *Server) AddStudyLabel(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	// Verify study exists.
	if _, err := model.GetStudyByID(r.Context(), s.db, id); err != nil {
		s.writeError(w, http.StatusNotFound, "study not found")
		return
	}

	var req addLabelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	label := strings.TrimSpace(req.Label)
	if label == "" {
		s.writeError(w, http.StatusBadRequest, "label is required")
		return
	}
	if len(label) > 80 {
		s.writeError(w, http.StatusBadRequest, "label must be 80 characters or fewer")
		return
	}

	l, err := model.AddStudyLabel(r.Context(), s.db, id, label, actorEmail(r))
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "create failed")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "study.label_added", actorEmail(r), "study", id, clientIP(r), map[string]any{
		"label": label,
	})
	s.writeJSON(w, http.StatusCreated, l)
}

// DeleteStudyLabel DELETE /api/studies/{id}/labels/{labelID}
func (s *Server) DeleteStudyLabel(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	labelID := r.PathValue("labelID")
	if err := model.DeleteStudyLabel(r.Context(), s.db, labelID, id); err != nil {
		s.writeError(w, http.StatusInternalServerError, "delete failed")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "study.label_removed", actorEmail(r), "study", id, clientIP(r), map[string]any{
		"label_id": labelID,
	})
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
