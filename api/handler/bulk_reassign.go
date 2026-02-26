package handler

import (
	"encoding/json"
	"net/http"

	"github.com/aegis-imaging/aegis/api/model"
)

type bulkReassignRequest struct {
	StudyIDs     []string `json:"study_ids"`
	NewProjectID string   `json:"new_project_id"`
}

// BulkReassignStudies POST /api/studies/bulk-reassign
func (s *Server) BulkReassignStudies(w http.ResponseWriter, r *http.Request) {
	var req bulkReassignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
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
	if req.NewProjectID == "" {
		s.writeError(w, http.StatusBadRequest, "new_project_id is required")
		return
	}

	actor := actorEmail(r)
	ip := clientIP(r)
	var processed int
	var errors []string
	for _, id := range req.StudyIDs {
		if err := model.ReassignStudyProject(r.Context(), s.db, id, req.NewProjectID); err != nil {
			errors = append(errors, id+": "+err.Error())
			continue
		}
		processed++
	}

	model.CreateAuditEntry(r.Context(), s.db, "study.bulk_reassigned", actor, "project", req.NewProjectID, ip, map[string]any{
		"study_count": len(req.StudyIDs), "processed": processed, "errors": len(errors),
	})

	s.writeJSON(w, http.StatusOK, map[string]any{
		"processed": processed,
		"errors":    errors,
	})
}
