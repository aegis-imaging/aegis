package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/aegis-imaging/aegis/api/model"
)

type bulkLabelRequest struct {
	StudyIDs []string `json:"study_ids"`
	Label    string   `json:"label"`
	Action   string   `json:"action"` // "add" or "remove"
}

// BulkLabelStudies POST /api/studies/bulk-label
// Applies or removes a label across multiple studies in one call.
func (s *Server) BulkLabelStudies(w http.ResponseWriter, r *http.Request) {
	var req bulkLabelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Label = strings.TrimSpace(req.Label)
	if req.Label == "" {
		s.writeError(w, http.StatusBadRequest, "label is required")
		return
	}
	if len(req.Label) > 80 {
		s.writeError(w, http.StatusBadRequest, "label must be 80 characters or fewer")
		return
	}
	if len(req.StudyIDs) == 0 {
		s.writeError(w, http.StatusBadRequest, "study_ids must not be empty")
		return
	}
	if len(req.StudyIDs) > 200 {
		s.writeError(w, http.StatusBadRequest, "bulk operations are limited to 200 studies at a time")
		return
	}

	actor := actorEmail(r)
	ip := clientIP(r)

	switch req.Action {
	case "add":
		n, err := model.BulkAddLabel(r.Context(), s.db, req.StudyIDs, req.Label, actor)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, "bulk label failed")
			return
		}
		model.CreateAuditEntry(r.Context(), s.db, "study.bulk_label_added", actor, "study", "", ip, map[string]any{
			"label": req.Label, "study_count": len(req.StudyIDs), "applied_count": n,
		})
		s.writeJSON(w, http.StatusOK, map[string]any{"applied": n, "total": len(req.StudyIDs)})
	case "remove":
		n, err := model.BulkRemoveLabel(r.Context(), s.db, req.StudyIDs, req.Label)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, "bulk unlabel failed")
			return
		}
		model.CreateAuditEntry(r.Context(), s.db, "study.bulk_label_removed", actor, "study", "", ip, map[string]any{
			"label": req.Label, "study_count": len(req.StudyIDs), "removed_count": n,
		})
		s.writeJSON(w, http.StatusOK, map[string]any{"removed": n, "total": len(req.StudyIDs)})
	default:
		s.writeError(w, http.StatusBadRequest, "action must be 'add' or 'remove'")
	}
}

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
