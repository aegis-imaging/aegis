package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/aegis-imaging/aegis/api/model"
)

type bulkCustomFieldRequest struct {
	StudyIDs []string `json:"study_ids"`
	FieldID  string   `json:"field_id"`
	Value    string   `json:"value"`
}

// BulkSetCustomField POST /api/studies/bulk-custom-field
// Sets a custom field value across multiple studies at once.
func (s *Server) BulkSetCustomField(w http.ResponseWriter, r *http.Request) {
	var req bulkCustomFieldRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.FieldID = strings.TrimSpace(req.FieldID)
	if req.FieldID == "" {
		s.writeError(w, http.StatusBadRequest, "field_id is required")
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

	applied := 0
	for _, sid := range req.StudyIDs {
		if err := model.UpsertStudyCustomFieldValue(r.Context(), s.db, sid, req.FieldID, req.Value); err == nil {
			applied++
		}
	}

	actor := actorEmail(r)
	model.CreateAuditEntry(r.Context(), s.db, "study.bulk_custom_field_set", actor, "study", "", clientIP(r), map[string]any{
		"field_id": req.FieldID, "study_count": len(req.StudyIDs), "applied": applied,
	})

	s.writeJSON(w, http.StatusOK, map[string]any{
		"applied": applied,
		"total":   len(req.StudyIDs),
	})
}
