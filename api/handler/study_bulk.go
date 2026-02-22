package handler

import (
	"encoding/json"
	"net/http"

	"github.com/aegis-imaging/aegis/api/email"
	"github.com/aegis-imaging/aegis/api/model"
)

type bulkStudyRequest struct {
	Action   string   `json:"action"`
	StudyIDs []string `json:"study_ids"`
}

type bulkStudyError struct {
	StudyID string `json:"study_id"`
	Error   string `json:"error"`
}

type bulkStudyResponse struct {
	Processed int              `json:"processed"`
	Errors    []bulkStudyError `json:"errors"`
}

// BulkStudyAction applies approve or reject to a list of study IDs.
func (s *Server) BulkStudyAction(w http.ResponseWriter, r *http.Request) {
	var req bulkStudyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Action != "approve" && req.Action != "reject" {
		s.writeError(w, http.StatusBadRequest, "action must be 'approve' or 'reject'")
		return
	}
	if len(req.StudyIDs) == 0 {
		s.writeError(w, http.StatusBadRequest, "study_ids must not be empty")
		return
	}
	if len(req.StudyIDs) > 200 {
		s.writeError(w, http.StatusBadRequest, "too many study_ids (max 200)")
		return
	}

	actor := actorEmail(r)
	ip := clientIP(r)
	resp := bulkStudyResponse{Errors: []bulkStudyError{}}

	for _, id := range req.StudyIDs {
		study, err := model.GetStudyByID(r.Context(), s.db, id)
		if err != nil {
			resp.Errors = append(resp.Errors, bulkStudyError{StudyID: id, Error: "not found"})
			continue
		}

		switch req.Action {
		case "approve":
			if study.Status == "approved" || study.Status == "rejected" {
				resp.Errors = append(resp.Errors, bulkStudyError{StudyID: id, Error: "already " + study.Status})
				continue
			}
			if err := model.UpdateStudyStatus(r.Context(), s.db, study.ID, "approved"); err != nil {
				resp.Errors = append(resp.Errors, bulkStudyError{StudyID: id, Error: "failed to update status"})
				continue
			}
			model.CreateAuditEntry(r.Context(), s.db, "study.approved", actor, "study", study.ID, ip, nil)
			if uploaderEmail, err := model.GetUploaderEmail(r.Context(), s.db, study.ID); err == nil && uploaderEmail != "" {
				subject, body := email.StudyApproved(study.StudyInstanceUID)
				if err := s.mailer.Send(r.Context(), uploaderEmail, subject, body); err != nil {
					// non-fatal
					_ = err
				}
			}
			if updated, err := model.GetStudyByID(r.Context(), s.db, study.ID); err == nil {
				if updated.ExportRequired && updated.ExportStatus == "pending" {
					go s.runExportForward(updated)
				}
			}

		case "reject":
			if study.Status == "rejected" {
				resp.Errors = append(resp.Errors, bulkStudyError{StudyID: id, Error: "already rejected"})
				continue
			}
			if err := model.UpdateStudyStatus(r.Context(), s.db, study.ID, "rejected"); err != nil {
				resp.Errors = append(resp.Errors, bulkStudyError{StudyID: id, Error: "failed to update status"})
				continue
			}
			model.CreateAuditEntry(r.Context(), s.db, "study.rejected", actor, "study", study.ID, ip, nil)
			if uploaderEmail, err := model.GetUploaderEmail(r.Context(), s.db, study.ID); err == nil && uploaderEmail != "" {
				subject, body := email.StudyRejected(study.StudyInstanceUID)
				if err := s.mailer.Send(r.Context(), uploaderEmail, subject, body); err != nil {
					// non-fatal
					_ = err
				}
			}
		}

		resp.Processed++
	}

	s.writeJSON(w, http.StatusOK, resp)
}
