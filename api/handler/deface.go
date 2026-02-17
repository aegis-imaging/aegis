package handler

import (
	"net/http"

	"github.com/msenjem/aegis/api/model"
)

func (s *Server) TriggerDeface(w http.ResponseWriter, r *http.Request) {
	studyUID := r.PathValue("studyUID")
	if studyUID == "" {
		s.writeError(w, http.StatusBadRequest, "missing study UID")
		return
	}

	study, err := model.GetStudyByUID(r.Context(), s.db, studyUID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "study not found")
		return
	}

	if err := model.UpdateStudyStatus(r.Context(), s.db, study.ID, "defacing"); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to update study status")
		return
	}

	model.CreateAuditEntry(r.Context(), s.db, "deface.triggered", "anonymous", "study", study.ID, clientIP(r), map[string]any{
		"study_uid": studyUID,
	})

	s.writeJSON(w, http.StatusAccepted, map[string]string{
		"status":  "defacing",
		"message": "Defacing pipeline triggered (stub — not yet implemented)",
	})
}
