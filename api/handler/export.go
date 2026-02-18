package handler

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/msenjem/aegis/api/email"
	"github.com/msenjem/aegis/api/model"
)

// ApproveStudy transitions a study to 'approved', making it eligible for export sharing.
func (s *Server) ApproveStudy(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	study, err := model.GetStudyByID(r.Context(), s.db, id)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "study not found")
		return
	}
	if study.Status == "approved" || study.Status == "rejected" {
		s.writeError(w, http.StatusBadRequest, "study is already "+study.Status)
		return
	}
	if err := model.UpdateStudyStatus(r.Context(), s.db, study.ID, "approved"); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to update status")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "study.approved", actorEmail(r), "study", study.ID, clientIP(r), nil)

	if uploaderEmail, err := model.GetUploaderEmail(r.Context(), s.db, study.ID); err == nil && uploaderEmail != "" {
		subject, body := email.StudyApproved(study.StudyInstanceUID)
		if err := s.mailer.Send(r.Context(), uploaderEmail, subject, body); err != nil {
			log.Printf("approve email to %s: %v", uploaderEmail, err)
		}
	}

	s.writeJSON(w, http.StatusOK, map[string]string{"status": "approved"})
}

// RejectStudy transitions a study to 'rejected'.
func (s *Server) RejectStudy(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	study, err := model.GetStudyByID(r.Context(), s.db, id)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "study not found")
		return
	}
	if study.Status == "rejected" {
		s.writeError(w, http.StatusBadRequest, "study is already rejected")
		return
	}
	if err := model.UpdateStudyStatus(r.Context(), s.db, study.ID, "rejected"); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to update status")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "study.rejected", actorEmail(r), "study", study.ID, clientIP(r), nil)

	if uploaderEmail, err := model.GetUploaderEmail(r.Context(), s.db, study.ID); err == nil && uploaderEmail != "" {
		subject, body := email.StudyRejected(study.StudyInstanceUID)
		if err := s.mailer.Send(r.Context(), uploaderEmail, subject, body); err != nil {
			log.Printf("reject email to %s: %v", uploaderEmail, err)
		}
	}

	s.writeJSON(w, http.StatusOK, map[string]string{"status": "rejected"})
}

type createShareRequest struct {
	RecipientEmail string `json:"recipient_email"`
	Note           string `json:"note"`
	ExpiryHours    int    `json:"expiry_hours"` // default 168 (7 days)
}

type createShareResponse struct {
	*model.ExportShare
	ExportURL string `json:"export_url"`
}

// CreateShare creates a time-limited, revocable export share for an approved study.
// The raw token is returned once in the response and never stored.
func (s *Server) CreateShare(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	study, err := model.GetStudyByID(r.Context(), s.db, id)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "study not found")
		return
	}
	if study.Status != "approved" {
		s.writeError(w, http.StatusBadRequest, "only approved studies can be shared")
		return
	}

	var req createShareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.RecipientEmail == "" {
		s.writeError(w, http.StatusBadRequest, "recipient_email is required")
		return
	}
	if req.ExpiryHours <= 0 {
		req.ExpiryHours = 168 // 7 days
	}

	rawToken, tokenHash, err := generateShareToken()
	if err != nil {
		log.Printf("generate share token: %v", err)
		s.writeError(w, http.StatusInternalServerError, "failed to generate share token")
		return
	}

	expiresAt := time.Now().Add(time.Duration(req.ExpiryHours) * time.Hour)
	share, err := model.CreateExportShare(r.Context(), s.db,
		study.ID, tokenHash, req.RecipientEmail, req.Note, actorEmail(r), expiresAt)
	if err != nil {
		log.Printf("create export share: %v", err)
		s.writeError(w, http.StatusInternalServerError, "failed to create share")
		return
	}

	share.Token = rawToken
	exportURL := fmt.Sprintf("%s/api/export/%s", s.cfg.APIBaseURL, rawToken)
	model.CreateAuditEntry(r.Context(), s.db, "share.created", actorEmail(r), "export_share", share.ID, clientIP(r), map[string]any{
		"recipient":  req.RecipientEmail,
		"study_id":   study.ID,
		"expires_at": expiresAt,
	})

	subject, body := email.ShareCreated(exportURL, share.ExpiresAt, share.Note)
	if err := s.mailer.Send(r.Context(), share.RecipientEmail, subject, body); err != nil {
		log.Printf("share email to %s: %v", share.RecipientEmail, err)
	}

	s.writeJSON(w, http.StatusCreated, createShareResponse{
		ExportShare: share,
		ExportURL:   exportURL,
	})
}

// ListShares returns all export shares for a study.
func (s *Server) ListShares(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	shares, err := model.ListExportSharesByStudy(r.Context(), s.db, id)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to list shares")
		return
	}
	if shares == nil {
		shares = []model.ExportShare{}
	}
	s.writeJSON(w, http.StatusOK, shares)
}

// RevokeShare immediately revokes an export share.
func (s *Server) RevokeShare(w http.ResponseWriter, r *http.Request) {
	shareID := r.PathValue("shareID")
	if err := model.RevokeExportShare(r.Context(), s.db, shareID); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to revoke share")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "share.revoked", actorEmail(r), "export_share", shareID, clientIP(r), nil)
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}

type exportFile struct {
	Key string `json:"key"`
	URL string `json:"url"`
}

type redeemResponse struct {
	ShareID   string       `json:"share_id"`
	StudyUID  string       `json:"study_uid"`
	Modality  string       `json:"modality"`
	ExpiresAt time.Time    `json:"expires_at"`
	Files     []exportFile `json:"files"`
}

// RedeemExport is the public (token-authenticated) endpoint for recipients to
// retrieve signed download URLs for a shared study's DICOM files.
func (s *Server) RedeemExport(w http.ResponseWriter, r *http.Request) {
	rawToken := r.PathValue("token")
	if rawToken == "" {
		s.writeError(w, http.StatusBadRequest, "missing token")
		return
	}

	h := sha256.Sum256([]byte(rawToken))
	tokenHash := hex.EncodeToString(h[:])

	share, err := model.GetExportShareByTokenHash(r.Context(), s.db, tokenHash)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "share not found")
		return
	}
	if share.RevokedAt != nil {
		s.writeError(w, http.StatusGone, "share has been revoked")
		return
	}
	if time.Now().After(share.ExpiresAt) {
		s.writeError(w, http.StatusGone, "share has expired")
		return
	}

	study, err := model.GetStudyByID(r.Context(), s.db, share.StudyID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "study not found")
		return
	}

	// List DICOM files in the study's current store.
	prefix := fmt.Sprintf("dicom/%s/%s", study.DicomStore, study.StudyInstanceUID)
	keys, err := s.store.List(r.Context(), prefix)
	if err != nil {
		log.Printf("list study files for export: %v", err)
		s.writeError(w, http.StatusInternalServerError, "failed to list study files")
		return
	}

	// Generate a signed download URL for each file (2-hour validity).
	files := make([]exportFile, 0, len(keys))
	for _, key := range keys {
		url, err := s.store.GenerateDownloadURL(r.Context(), key, 2*time.Hour)
		if err != nil {
			log.Printf("generate download URL for %s: %v", key, err)
			continue
		}
		files = append(files, exportFile{Key: key, URL: url})
	}

	// Log the redemption event.
	model.CreateExportDownload(r.Context(), s.db, share.ID, clientIP(r))
	model.CreateAuditEntry(r.Context(), s.db, "export.redeemed", share.RecipientEmail, "export_share", share.ID, clientIP(r), map[string]any{
		"study_uid":  study.StudyInstanceUID,
		"file_count": len(files),
	})

	s.writeJSON(w, http.StatusOK, redeemResponse{
		ShareID:   share.ID,
		StudyUID:  study.StudyInstanceUID,
		Modality:  study.Modality,
		ExpiresAt: share.ExpiresAt,
		Files:     files,
	})
}

// generateShareToken creates a cryptographically random 32-byte token and returns
// both the raw base64url-encoded token and its SHA-256 hex hash for storage.
func generateShareToken() (raw, hash string, err error) {
	buf := make([]byte, 32)
	if _, err = rand.Read(buf); err != nil {
		return
	}
	raw = base64.RawURLEncoding.EncodeToString(buf)
	h := sha256.Sum256([]byte(raw))
	hash = hex.EncodeToString(h[:])
	return
}
