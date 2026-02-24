package handler

import (
	"encoding/json"
	"net/http"

	"github.com/aegis-imaging/aegis/api/model"
)

// ReassignStudy PUT /api/studies/{id}/project
// Moves a study to a different project. Admin-only.
func (s *Server) ReassignStudy(w http.ResponseWriter, r *http.Request) {
	studyID := r.PathValue("id")

	study, err := model.GetStudyByID(r.Context(), s.db, studyID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "study not found")
		return
	}

	var req struct {
		ProjectID string `json:"project_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.ProjectID == "" {
		s.writeError(w, http.StatusBadRequest, "project_id is required")
		return
	}
	if req.ProjectID == study.ProjectID {
		s.writeError(w, http.StatusBadRequest, "study is already in this project")
		return
	}

	// Verify the target project exists.
	targetProject, err := model.GetProjectByID(r.Context(), s.db, req.ProjectID)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "target project not found")
		return
	}

	oldProjectID := study.ProjectID
	if err := model.ReassignStudyProject(r.Context(), s.db, studyID, req.ProjectID); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to reassign study")
		return
	}

	model.CreateAuditEntry(r.Context(), s.db, "study.reassigned", actorEmail(r), "study", studyID, clientIP(r), map[string]any{
		"from_project_id": oldProjectID,
		"to_project_id":   req.ProjectID,
		"to_project_name": targetProject.Name,
	})

	study.ProjectID = req.ProjectID
	s.writeJSON(w, http.StatusOK, study)
}
