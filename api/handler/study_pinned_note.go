package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/aegis-imaging/aegis/api/model"
)

// ListStudyPinnedNotes GET /api/studies/{id}/pinned-notes
func (s *Server) ListStudyPinnedNotes(w http.ResponseWriter, r *http.Request) {
	studyID := r.PathValue("id")
	notes, err := model.ListStudyPinnedNotes(r.Context(), s.db, studyID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	if notes == nil {
		notes = []model.StudyPinnedNote{}
	}
	s.writeJSON(w, http.StatusOK, notes)
}

// CreateStudyPinnedNote POST /api/studies/{id}/pinned-notes
func (s *Server) CreateStudyPinnedNote(w http.ResponseWriter, r *http.Request) {
	studyID := r.PathValue("id")

	// Verify study exists.
	if _, err := model.GetStudyByID(r.Context(), s.db, studyID); err != nil {
		s.writeError(w, http.StatusNotFound, "study not found")
		return
	}

	var req struct {
		Body string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Body = strings.TrimSpace(req.Body)
	if req.Body == "" {
		s.writeError(w, http.StatusBadRequest, "body is required")
		return
	}
	if len(req.Body) > 2000 {
		s.writeError(w, http.StatusBadRequest, "body must be 2000 characters or fewer")
		return
	}

	n := &model.StudyPinnedNote{
		StudyID: studyID,
		Author:  actorEmail(r),
		Body:    req.Body,
	}
	if err := model.CreateStudyPinnedNote(r.Context(), s.db, n); err != nil {
		s.writeError(w, http.StatusInternalServerError, "create failed")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "study.pinned_note_added", actorEmail(r), "study", studyID, clientIP(r), nil)
	s.writeJSON(w, http.StatusCreated, n)
}

// UpdateStudyPinnedNote PATCH /api/studies/{id}/pinned-notes/{noteID}
func (s *Server) UpdateStudyPinnedNote(w http.ResponseWriter, r *http.Request) {
	studyID := r.PathValue("id")
	noteID := r.PathValue("noteID")

	var req struct {
		Pinned *bool `json:"pinned"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Pinned == nil {
		s.writeError(w, http.StatusBadRequest, "pinned is required")
		return
	}

	if err := model.UpdateStudyPinnedNote(r.Context(), s.db, noteID, *req.Pinned); err != nil {
		s.writeError(w, http.StatusInternalServerError, "update failed")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "study.pinned_note_updated", actorEmail(r), "study", studyID, clientIP(r), map[string]any{
		"note_id": noteID, "pinned": *req.Pinned,
	})
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// DeleteStudyPinnedNote DELETE /api/studies/{id}/pinned-notes/{noteID}
func (s *Server) DeleteStudyPinnedNote(w http.ResponseWriter, r *http.Request) {
	studyID := r.PathValue("id")
	noteID := r.PathValue("noteID")
	if err := model.DeleteStudyPinnedNote(r.Context(), s.db, noteID, studyID); err != nil {
		s.writeError(w, http.StatusInternalServerError, "delete failed")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "study.pinned_note_removed", actorEmail(r), "study", studyID, clientIP(r), map[string]any{
		"note_id": noteID,
	})
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
