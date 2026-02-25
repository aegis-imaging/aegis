package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/aegis-imaging/aegis/api/model"
)

type studyNote struct {
	ID        string          `json:"id"`
	Actor     string          `json:"actor"`
	Note      string          `json:"note"`
	CreatedAt time.Time       `json:"created_at"`
}

// ListStudyNotes returns all admin notes recorded for a study (stored as
// study.note audit entries). Notes are ordered newest-first.
func (s *Server) ListStudyNotes(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := model.GetStudyByID(r.Context(), s.db, id); err != nil {
		s.writeError(w, http.StatusNotFound, "study not found")
		return
	}

	entries, err := model.ListStudyNoteAuditEntries(r.Context(), s.db, id)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to list notes")
		return
	}

	notes := make([]studyNote, 0, len(entries))
	for _, e := range entries {
		// Extract the "note" field from the JSONB detail.
		var detail struct {
			Note string `json:"note"`
		}
		if len(e.Detail) > 0 {
			_ = json.Unmarshal(e.Detail, &detail)
		}
		notes = append(notes, studyNote{
			ID:        e.ID,
			Actor:     e.Actor,
			Note:      detail.Note,
			CreatedAt: e.CreatedAt,
		})
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"study_id": id,
		"notes":    notes,
		"total":    len(notes),
	})
}

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
