package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/aegis-imaging/aegis/api/model"
)

type bulkStatusRequest struct {
	StudyIDs []string `json:"study_ids"`
	Status   string   `json:"status"`
}

// BulkStatusUpdate POST /api/studies/bulk-status
// Changes the status of multiple studies at once.
func (s *Server) BulkStatusUpdate(w http.ResponseWriter, r *http.Request) {
	var req bulkStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Status = strings.TrimSpace(req.Status)
	validStatuses := map[string]bool{"received": true, "approved": true, "rejected": true}
	if !validStatuses[req.Status] {
		s.writeError(w, http.StatusBadRequest, "status must be received, approved, or rejected")
		return
	}
	if len(req.StudyIDs) == 0 {
		s.writeError(w, http.StatusBadRequest, "study_ids must not be empty")
		return
	}
	if len(req.StudyIDs) > 200 {
		s.writeError(w, http.StatusBadRequest, "bulk operations limited to 200 studies")
		return
	}

	actor := actorEmail(r)
	ip := clientIP(r)
	updated := 0
	for _, sid := range req.StudyIDs {
		if err := model.UpdateStudyStatus(r.Context(), s.db, sid, req.Status); err == nil {
			updated++
		}
	}

	model.CreateAuditEntry(r.Context(), s.db, "study.bulk_status_updated", actor, "study", "", ip, map[string]any{
		"status": req.Status, "study_count": len(req.StudyIDs), "updated": updated,
	})

	s.writeJSON(w, http.StatusOK, map[string]any{
		"updated": updated,
		"total":   len(req.StudyIDs),
	})
}
