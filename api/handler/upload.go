package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/msenjem/aegis/api/email"
	"github.com/msenjem/aegis/api/model"
	"github.com/msenjem/aegis/api/routing"
)

type uploadInitRequest struct {
	ProjectSlug   string        `json:"project_slug"`
	FileCount     int           `json:"file_count"`
	UploaderEmail string        `json:"uploader_email"`
	Metadata      studyMetadata `json:"study_metadata"`
}

type studyMetadata struct {
	StudyInstanceUID string `json:"study_instance_uid"`
	Modality         string `json:"modality"`
	BodyPart         string `json:"body_part"`
	StudyDescription string `json:"study_description"`
	SeriesCount      int    `json:"series_count"`
	InstanceCount    int    `json:"instance_count"`
}

type uploadInitResponse struct {
	SessionID  string   `json:"session_id"`
	UploadURLs []string `json:"upload_urls"`
	ExpiresAt  string   `json:"expires_at"`
}

func (s *Server) UploadInit(w http.ResponseWriter, r *http.Request) {
	var req uploadInitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if req.FileCount <= 0 {
		s.writeError(w, http.StatusBadRequest, "file_count must be positive")
		return
	}

	// Look up project (default to "default")
	slug := req.ProjectSlug
	if slug == "" {
		slug = "default"
	}
	project, err := model.GetProjectBySlug(r.Context(), s.db, slug)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "project not found: "+slug)
		return
	}

	// Create upload session
	session, err := model.CreateUploadSession(r.Context(), s.db, project.ID, req.FileCount, "", clientIP(r), req.UploaderEmail)
	if err != nil {
		log.Printf("create upload session: %v", err)
		s.writeError(w, http.StatusInternalServerError, "failed to create upload session")
		return
	}

	// Set storage prefix using session ID
	prefix := fmt.Sprintf("uploads/%s", session.ID)

	// Generate upload URLs
	urls := make([]string, req.FileCount)
	for i := 0; i < req.FileCount; i++ {
		key := fmt.Sprintf("%s/%d.dcm", prefix, i)
		url, err := s.store.GenerateUploadURL(r.Context(), key, 1*time.Hour)
		if err != nil {
			log.Printf("generate upload URL: %v", err)
			s.writeError(w, http.StatusInternalServerError, "failed to generate upload URLs")
			return
		}
		urls[i] = url
	}

	// Audit
	model.CreateAuditEntry(r.Context(), s.db, "upload.init", "anonymous", "upload_session", session.ID, clientIP(r), map[string]any{
		"file_count": req.FileCount,
		"project":    slug,
	})

	s.writeJSON(w, http.StatusOK, uploadInitResponse{
		SessionID:  session.ID,
		UploadURLs: urls,
		ExpiresAt:  time.Now().Add(1 * time.Hour).UTC().Format(time.RFC3339),
	})
}

// UploadFile receives a single DICOM file for local dev mode.
// In production, files go directly to GCS via signed URLs.
func (s *Server) UploadFile(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("sessionID")
	index := r.PathValue("index")

	if sessionID == "" || index == "" {
		s.writeError(w, http.StatusBadRequest, "missing session ID or file index")
		return
	}

	// Validate session exists
	_, err := model.GetUploadSession(r.Context(), s.db, sessionID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "upload session not found")
		return
	}

	key := fmt.Sprintf("uploads/%s/%s.dcm", sessionID, index)
	if err := s.store.Store(r.Context(), key, r.Body); err != nil {
		log.Printf("store file: %v", err)
		s.writeError(w, http.StatusInternalServerError, "failed to store file")
		return
	}

	w.WriteHeader(http.StatusOK)
}

type uploadCompleteRequest struct {
	SessionID string `json:"session_id"`
}

type uploadCompleteResponse struct {
	SessionID string       `json:"session_id"`
	Status    string       `json:"status"`
	Study     *model.Study `json:"study"`
}

func (s *Server) UploadComplete(w http.ResponseWriter, r *http.Request) {
	var req uploadCompleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	session, err := model.GetUploadSession(r.Context(), s.db, req.SessionID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "upload session not found")
		return
	}

	if session.Status != "initiated" {
		s.writeError(w, http.StatusConflict, "upload session already processed")
		return
	}

	// Update status to ingesting
	if err := model.UpdateUploadSessionStatus(r.Context(), s.db, session.ID, "ingesting"); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to update session status")
		return
	}

	// List uploaded files
	prefix := fmt.Sprintf("uploads/%s", session.ID)
	files, err := s.store.List(r.Context(), prefix)
	if err != nil || len(files) == 0 {
		model.UpdateUploadSessionFailed(r.Context(), s.db, session.ID, "no files found in staging")
		s.writeError(w, http.StatusBadRequest, "no files found in staging area")
		return
	}

	// "Ingest" — move files from staging to DICOM store directory
	study, err := s.ingestFiles(r.Context(), session, files)
	if err != nil {
		model.UpdateUploadSessionFailed(r.Context(), s.db, session.ID, err.Error())
		s.writeError(w, http.StatusInternalServerError, "ingest failed: "+err.Error())
		return
	}

	if session.UploaderEmail != "" {
		subject, body := email.UploadConfirmed(study.StudyInstanceUID, study.Modality, len(files), study.CreatedAt)
		if err := s.mailer.Send(r.Context(), session.UploaderEmail, subject, body); err != nil {
			log.Printf("upload confirm email to %s: %v", session.UploaderEmail, err)
		}
	}

	// Audit
	model.CreateAuditEntry(r.Context(), s.db, "upload.complete", "anonymous", "study", study.ID, clientIP(r), map[string]any{
		"session_id":        session.ID,
		"study_uid":         study.StudyInstanceUID,
		"modality":          study.Modality,
		"instance_count":    len(files),
		"defacing_required": study.DefacingRequired,
	})

	s.writeJSON(w, http.StatusOK, uploadCompleteResponse{
		SessionID: session.ID,
		Status:    "completed",
		Study:     study,
	})
}

func (s *Server) ingestFiles(ctx context.Context, session *model.UploadSession, files []string) (*model.Study, error) {
	// Generate a study UID from session ID for local dev
	// In production, this would come from parsing DICOM headers
	studyUID := fmt.Sprintf("2.25.%s", strings.ReplaceAll(session.ID, "-", ""))

	// Move files from staging to DICOM store
	dstPrefix := fmt.Sprintf("dicom/raw/%s", studyUID)
	for i, srcKey := range files {
		dstKey := fmt.Sprintf("%s/%d.dcm", dstPrefix, i)
		if err := s.store.Move(ctx, srcKey, dstKey); err != nil {
			return nil, fmt.Errorf("move file %d: %w", i, err)
		}
	}

	// Clean up empty staging directory
	stagingDir := s.store.KeyToPath(fmt.Sprintf("uploads/%s", session.ID))
	if stagingDir != "" {
		os.Remove(stagingDir)
	}

	// Determine if defacing is needed
	bodyPart := strings.ToUpper(deref(session.BodyPart))
	defacingRequired := bodyPart == "HEAD" || bodyPart == "BRAIN"

	// Create study record
	study := &model.Study{
		ProjectID:        session.ProjectID,
		UploadSessionID:  &session.ID,
		StudyInstanceUID: studyUID,
		Modality:         deref(session.Modality),
		BodyPart:         deref(session.BodyPart),
		StudyDescription: "",
		SeriesCount:      0,
		InstanceCount:    len(files),
		Status:           "received",
		DefacingRequired: defacingRequired,
		DicomStore:       "raw",
		Source:           "external",
	}
	if err := model.CreateStudy(ctx, s.db, study); err != nil {
		return nil, fmt.Errorf("create study: %w", err)
	}

	// Evaluate routing rules — may mutate study (e.g. auto_approve, require_defacing).
	routing.EvaluateRules(ctx, s.db, s.store, study)

	// Auto-dispatch processing pipeline (classification → defacing → QC → BIDS, etc.)
	s.AdvancePipeline(ctx, study.ID)

	// Update session
	model.UpdateUploadSessionComplete(ctx, s.db, session.ID, studyUID, study.Modality, study.BodyPart)

	return study, nil
}

func clientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return strings.Split(fwd, ",")[0]
	}
	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		return ip[:idx]
	}
	return ip
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// storeDir returns the absolute path of a DICOM store directory.
func (s *Server) storeDir(storeName string) string {
	return filepath.Join(s.cfg.LocalStorageDir, "dicom", storeName)
}

// listDicomFiles lists DICOM files in a study directory.
func (s *Server) listDicomFiles(storeName, studyUID string) ([]string, error) {
	dir := filepath.Join(s.storeDir(storeName), studyUID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".dcm") {
			paths = append(paths, filepath.Join(dir, e.Name()))
		}
	}
	return paths, nil
}

// serveDicomFile serves a single DICOM file as application/dicom.
func (s *Server) serveDicomFile(w http.ResponseWriter, path string) {
	f, err := os.Open(path)
	if err != nil {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}
	defer f.Close()
	w.Header().Set("Content-Type", "application/dicom")
	io.Copy(w, f)
}
