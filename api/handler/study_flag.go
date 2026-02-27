package handler

import (
	"encoding/json"
	"net/http"

	"github.com/aegis-imaging/aegis/api/model"
)

type patchStudyFlagRequest struct {
	Flagged bool `json:"flagged"`
}

// PatchStudyFlag sets or clears the priority_flag on a study.
//
// PATCH /api/studies/{id}/flag
func (s *Server) PatchStudyFlag(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req patchStudyFlagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if _, _, ok := s.requireStudyWriteAccessByID(w, r, id, projectWriteIntentStudyMutation); !ok {
		return
	}

	if err := model.SetPriorityFlag(r.Context(), s.db, id, req.Flagged); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to update flag")
		return
	}

	action := "study.flag_set"
	if !req.Flagged {
		action = "study.flag_cleared"
	}
	model.CreateAuditEntry(r.Context(), s.db, action, actorEmail(r), "study", id, clientIP(r), map[string]any{
		"flagged": req.Flagged,
	})

	s.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "flagged": req.Flagged})
}
