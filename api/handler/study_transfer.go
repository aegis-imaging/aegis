package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/aegis-imaging/aegis/api/model"
)

// ListStudyTransfers GET /api/studies/{id}/transfers
func (s *Server) ListStudyTransfers(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	transfers, err := model.ListStudyTransfers(r.Context(), s.db, id)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	if transfers == nil {
		transfers = []model.StudyTransfer{}
	}
	s.writeJSON(w, http.StatusOK, transfers)
}

// TransferStudy POST /api/studies/{id}/transfer
// Moves a study to a different project and records the transfer.
func (s *Server) TransferStudy(w http.ResponseWriter, r *http.Request) {
	studyID := r.PathValue("id")

	var req struct {
		ToProjectID string `json:"to_project_id"`
		Reason      string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.ToProjectID = strings.TrimSpace(req.ToProjectID)
	if req.ToProjectID == "" {
		s.writeError(w, http.StatusBadRequest, "to_project_id is required")
		return
	}

	// Get current study to record from_project_id.
	study, err := model.GetStudyByID(r.Context(), s.db, studyID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "study not found")
		return
	}
	if study.ProjectID == req.ToProjectID {
		s.writeError(w, http.StatusBadRequest, "study is already in the target project")
		return
	}

	// Verify target project exists.
	if _, err := model.GetProjectByID(r.Context(), s.db, req.ToProjectID); err != nil {
		s.writeError(w, http.StatusBadRequest, "target project not found")
		return
	}

	actor := actorEmail(r)
	ip := clientIP(r)

	// Record the transfer.
	t := &model.StudyTransfer{
		StudyID:       studyID,
		FromProjectID: study.ProjectID,
		ToProjectID:   req.ToProjectID,
		TransferredBy: actor,
		Reason:        strings.TrimSpace(req.Reason),
	}
	if err := model.CreateStudyTransfer(r.Context(), s.db, t); err != nil {
		s.writeError(w, http.StatusInternalServerError, "record transfer failed")
		return
	}

	// Move the study.
	if err := model.ReassignStudyProject(r.Context(), s.db, studyID, req.ToProjectID); err != nil {
		s.writeError(w, http.StatusInternalServerError, "transfer failed")
		return
	}

	model.CreateAuditEntry(r.Context(), s.db, "study.transferred", actor, "study", studyID, ip, map[string]any{
		"from_project_id": study.ProjectID,
		"to_project_id":   req.ToProjectID,
		"reason":          t.Reason,
	})

	s.writeJSON(w, http.StatusOK, t)
}
