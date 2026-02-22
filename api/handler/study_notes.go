package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/aegis-imaging/aegis/api/model"
)

type addNoteRequest struct {
	Note string `json:"note"`
}

// AddStudyNote stores an internal admin note on a study as an audit entry.
func (s *Server) AddStudyNote(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req addNoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	note := strings.TrimSpace(req.Note)
	if note == "" {
		s.writeError(w, http.StatusBadRequest, "note must not be empty")
		return
	}
	if len(note) > 2000 {
		s.writeError(w, http.StatusBadRequest, "note too long (max 2000 characters)")
		return
	}

	if _, err := model.GetStudyByID(r.Context(), s.db, id); err != nil {
		s.writeError(w, http.StatusNotFound, "study not found")
		return
	}

	model.CreateAuditEntry(r.Context(), s.db, "study.note", actorEmail(r), "study", id, clientIP(r), map[string]any{
		"note": note,
	})

	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
