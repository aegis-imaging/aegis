package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/aegis-imaging/aegis/api/email"
	"github.com/aegis-imaging/aegis/api/model"
)

type bulkShareRequest struct {
	StudyIDs      []string `json:"study_ids"`
	RecipientEmail string  `json:"recipient_email"`
	Note          string   `json:"note"`
	ExpiryHours   int      `json:"expiry_hours"`   // default 168 (7 days)
	MaxDownloads  *int     `json:"max_downloads,omitempty"`
}

type bulkShareError struct {
	StudyID string `json:"study_id"`
	Error   string `json:"error"`
}

type bulkShareResponse struct {
	Created int              `json:"created"`
	Errors  []bulkShareError `json:"errors"`
}

// BulkCreateShares creates export shares for multiple studies in one call.
//
// POST /api/studies/bulk-share
// Body: {"study_ids":["<uuid>",...], "recipient_email":"...", "expiry_hours":168, "note":"..."}
// Returns: {"created": N, "errors": [{"study_id":"...","error":"..."}]}
func (s *Server) BulkCreateShares(w http.ResponseWriter, r *http.Request) {
	var req bulkShareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if len(req.StudyIDs) == 0 {
		s.writeError(w, http.StatusBadRequest, "study_ids must not be empty")
		return
	}
	if len(req.StudyIDs) > 200 {
		s.writeError(w, http.StatusBadRequest, "study_ids must not exceed 200 per call")
		return
	}
	if req.RecipientEmail == "" {
		s.writeError(w, http.StatusBadRequest, "recipient_email is required")
		return
	}
	if req.MaxDownloads != nil && *req.MaxDownloads <= 0 {
		s.writeError(w, http.StatusBadRequest, "max_downloads must be a positive integer")
		return
	}

	// Resolve expiry once for all shares in the batch.
	proxied := createShareRequest{
		RecipientEmail: req.RecipientEmail,
		Note:           req.Note,
		ExpiryHours:    req.ExpiryHours,
		MaxDownloads:   req.MaxDownloads,
	}
	expiresAt, err := resolveShareExpiry(proxied, time.Now().UTC())
	if err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	actor := actorEmail(r)
	ip := clientIP(r)

	var created int
	var errs []bulkShareError

	for _, studyID := range req.StudyIDs {
		study, err := model.GetStudyByID(r.Context(), s.db, studyID)
		if err != nil {
			errs = append(errs, bulkShareError{StudyID: studyID, Error: "study not found"})
			continue
		}
		if study.Status != "approved" {
			errs = append(errs, bulkShareError{StudyID: studyID, Error: "study is not approved"})
			continue
		}

		rawToken, tokenHash, err := generateShareToken()
		if err != nil {
			log.Printf("bulk share token gen for %s: %v", studyID, err)
			errs = append(errs, bulkShareError{StudyID: studyID, Error: "failed to generate token"})
			continue
		}

		share, err := model.CreateExportShare(r.Context(), s.db,
			study.ID, tokenHash, req.RecipientEmail, req.Note, actor, expiresAt, req.MaxDownloads)
		if err != nil {
			log.Printf("bulk share create for %s: %v", studyID, err)
			errs = append(errs, bulkShareError{StudyID: studyID, Error: "failed to create share"})
			continue
		}

		var exportURL string
		if s.cfg.ExportPortalBaseURL != "" {
			exportURL = fmt.Sprintf("%s?token=%s", s.cfg.ExportPortalBaseURL, rawToken)
		} else {
			exportURL = fmt.Sprintf("%s/api/export/%s", s.cfg.APIBaseURL, rawToken)
		}

		model.CreateAuditEntry(r.Context(), s.db, "share.created", actor, "export_share", share.ID, ip, map[string]any{
			"recipient":  req.RecipientEmail,
			"study_id":   study.ID,
			"expires_at": expiresAt,
			"bulk":       true,
		})

		subject, body := email.ShareCreated(exportURL, share.ExpiresAt, share.Note)
		if err := s.mailer.Send(r.Context(), req.RecipientEmail, subject, body); err != nil {
			log.Printf("bulk share email to %s for %s: %v", req.RecipientEmail, studyID, err)
		}

		created++
	}

	if errs == nil {
		errs = []bulkShareError{}
	}

	s.writeJSON(w, http.StatusCreated, bulkShareResponse{
		Created: created,
		Errors:  errs,
	})
}
