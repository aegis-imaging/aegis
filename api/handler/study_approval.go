package handler

import (
	"net/http"

	"github.com/aegis-imaging/aegis/api/model"
)

// ListStudyApprovals GET /api/studies/{id}/approvals
// Returns the sign-off history for a study (who approved/rejected and when).
func (s *Server) ListStudyApprovals(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	approvals, err := model.ListStudyApprovals(r.Context(), s.db, id)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	if approvals == nil {
		approvals = []model.StudyApproval{}
	}
	s.writeJSON(w, http.StatusOK, approvals)
}

// RecordStudyApproval POST /api/studies/{id}/approvals
// Records a sign-off entry for a study (does not change study status — use approve/reject for that).
func (s *Server) RecordStudyApproval(w http.ResponseWriter, r *http.Request) {
	studyID := r.PathValue("id")

	// Verify study exists.
	if _, err := model.GetStudyByID(r.Context(), s.db, studyID); err != nil {
		s.writeError(w, http.StatusNotFound, "study not found")
		return
	}

	actor := actorEmail(r)
	a := &model.StudyApproval{
		StudyID:    studyID,
		ApprovedBy: actor,
		Action:     "signed_off",
	}
	if err := model.CreateStudyApproval(r.Context(), s.db, a); err != nil {
		s.writeError(w, http.StatusInternalServerError, "create failed")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "study.signed_off", actor, "study", studyID, clientIP(r), nil)
	s.writeJSON(w, http.StatusCreated, a)
}
