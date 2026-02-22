package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/aegis-imaging/aegis/api/model"
)

// ListSubjects GET /api/subjects?project_id=<uuid>
// Returns unique subject IDs with study counts.
func (s *Server) ListSubjects(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project_id")
	subjects, err := model.ListSubjects(r.Context(), s.db, projectID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	if subjects == nil {
		subjects = []model.SubjectSummary{}
	}
	s.writeJSON(w, http.StatusOK, subjects)
}

type setSubjectRequest struct {
	SubjectID string `json:"subject_id"` // empty string clears the subject ID
}

// SetStudySubject PUT /api/studies/{id}/subject
// Sets or clears the subject_id on a study.
func (s *Server) SetStudySubject(w http.ResponseWriter, r *http.Request) {
	studyID := r.PathValue("id")

	study, err := model.GetStudyByID(r.Context(), s.db, studyID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "study not found")
		return
	}

	var req setSubjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.SubjectID = strings.TrimSpace(req.SubjectID)

	var subjectID *string
	if req.SubjectID != "" {
		subjectID = &req.SubjectID
	}

	if err := model.UpdateStudySubjectID(r.Context(), s.db, studyID, subjectID); err != nil {
		s.writeError(w, http.StatusInternalServerError, "update failed")
		return
	}

	model.CreateAuditEntry(r.Context(), s.db, "study.subject_set", actorEmail(r), "study", studyID, clientIP(r), map[string]any{
		"subject_id": req.SubjectID,
	})

	study.SubjectID = subjectID
	s.writeJSON(w, http.StatusOK, study)
}
