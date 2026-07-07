package handler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	dicomlib "github.com/suyashkumar/dicom"
	"github.com/suyashkumar/dicom/pkg/tag"

	"github.com/aegis-imaging/aegis/api/email"
	"github.com/aegis-imaging/aegis/api/middleware"
	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/routing"
	"github.com/aegis-imaging/aegis/api/storage"
	"github.com/aegis-imaging/aegis/api/webhook"
)

type uploadInitRequest struct {
	ProjectSlug        string        `json:"project_slug"`
	InstitutionID      string        `json:"institution_id,omitempty"`
	InstitutionSlug    string        `json:"institution_slug,omitempty"`
	InstitutionAETitle string        `json:"institution_ae_title,omitempty"`
	FileCount          int           `json:"file_count"`
	UploaderEmail      string        `json:"uploader_email"`
	Metadata           studyMetadata `json:"study_metadata"`
}

type seriesMetadata struct {
	SeriesInstanceUID string `json:"series_instance_uid"`
	SeriesDescription string `json:"series_description"`
	Modality          string `json:"modality"`
	BodyPart          string `json:"body_part"`
	InstanceCount     int    `json:"instance_count"`
}

type studyMetadata struct {
	StudyInstanceUID string           `json:"study_instance_uid"`
	Modality         string           `json:"modality"`
	BodyPart         string           `json:"body_part"`
	StudyDescription string           `json:"study_description"`
	StudyDate        string           `json:"study_date"`
	SeriesCount      int              `json:"series_count"`
	InstanceCount    int              `json:"instance_count"`
	StudySizeBytes   int64            `json:"study_size_bytes,omitempty"`
	Series           []seriesMetadata `json:"series,omitempty"`
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

	// Per-project authorization for signed-in upload-portal users. Only applies
	// when auth is enabled and the caller authenticated via the uploader session
	// cookie — satellites are attributed by cert (project link is checked at
	// routing time) and API keys are platform-scoped. Any project_members role
	// qualifies; the dedicated upload role is 'uploader'.
	if s.cfg.AuthEnabled && middleware.SatelliteFromContext(r.Context()) == nil {
		if sess := middleware.UploaderSessionFromContext(r.Context()); sess != nil {
			user := middleware.UserFromContext(r.Context())
			if _, err := loadProjectMemberRoleForUploader(r, s.db, project.ID, user.ID); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					s.writeError(w, http.StatusForbidden, "you don't have upload access to this project")
					return
				}
				log.Printf("upload init: project access check for %s on %s: %v", user.ID, project.ID, err)
				s.writeError(w, http.StatusInternalServerError, "failed to verify project access")
				return
			}
		}
	}

	var institutionID *string
	// Satellite-mTLS attribution wins over body selectors and IP allowlist — the
	// cert is the strongest identity signal we have.
	if satellite := middleware.SatelliteFromContext(r.Context()); satellite != nil && satellite.Institution != nil {
		institutionID = &satellite.Institution.ID
	} else if strings.TrimSpace(req.InstitutionID) != "" || strings.TrimSpace(req.InstitutionSlug) != "" || strings.TrimSpace(req.InstitutionAETitle) != "" {
		inst, err := s.resolveIngestInstitution(r.Context(), project.ID, req.InstitutionID, req.InstitutionSlug, req.InstitutionAETitle)
		if err != nil {
			s.writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		institutionID = &inst.ID
	} else {
		inst, err := s.resolveIngestInstitutionFromSourceIP(r.Context(), project.ID, clientIP(r))
		if err == nil && inst != nil {
			institutionID = &inst.ID
		}
	}

	// Browser upload allowlist (chunk 2). No-op when no institution was
	// attributed — legacy/anonymous uploads keep working as before.
	if !s.enforceUploadMethod(w, r, institutionID, "browser.web-upload") {
		return
	}

	// Create upload session
	session, err := model.CreateUploadSession(r.Context(), s.db, project.ID, req.FileCount, "", clientIP(r), req.UploaderEmail)
	if err != nil {
		log.Printf("create upload session: %v", err)
		s.writeError(w, http.StatusInternalServerError, "failed to create upload session")
		return
	}
	if err := model.UpdateUploadSessionInstitution(r.Context(), s.db, session.ID, institutionID); err != nil {
		log.Printf("set upload session institution: %v", err)
		s.writeError(w, http.StatusInternalServerError, "failed to initialize upload session")
		return
	}
	session.InstitutionID = institutionID

	// Store client-provided metadata on the session for later use at upload-complete.
	if req.Metadata.Modality != "" || req.Metadata.BodyPart != "" || req.Metadata.StudyDate != "" {
		var mod, bp, sd *string
		if req.Metadata.Modality != "" {
			mod = &req.Metadata.Modality
		}
		if req.Metadata.BodyPart != "" {
			bp = &req.Metadata.BodyPart
		}
		if req.Metadata.StudyDate != "" {
			sd = &req.Metadata.StudyDate
		}
		if err := model.UpdateUploadSessionMetadata(r.Context(), s.db, session.ID, mod, bp, sd); err != nil {
			log.Printf("set upload session metadata: %v", err)
		}
		session.Modality = mod
		session.BodyPart = bp
		session.StudyDate = sd
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
	model.CreateAuditEntry(r.Context(), s.db, "upload.init", uploadActor(r), "upload_session", session.ID, clientIP(r), map[string]any{
		"file_count":     req.FileCount,
		"project":        slug,
		"institution_id": institutionID,
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

	// Enforce per-project storage quota (if set).
	if err := s.checkStorageQuota(r.Context(), session.ProjectID); err != nil {
		s.writeError(w, http.StatusRequestEntityTooLarge, err.Error())
		return
	}

	// Atomically claim the session (initiated -> ingesting). Two concurrent
	// completes race on this single UPDATE; the loser gets 409 instead of
	// double-ingesting the same staging files.
	claimed, err := model.MarkUploadSessionIngesting(r.Context(), s.db, session.ID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to update session status")
		return
	}
	if !claimed {
		s.writeError(w, http.StatusConflict, "upload session already completed or completing")
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

	// Default source for web uploads is "external"; satellite-mTLS overrides it
	// so studies are attributed to satellite-sourced traffic in routing/analytics.
	source := "external"
	if satellite := middleware.SatelliteFromContext(r.Context()); satellite != nil && satellite.Institution != nil {
		source = "satellite"
		// Satellite wins over a stale institution selector on the session row too.
		instID := satellite.Institution.ID
		if session.InstitutionID == nil || *session.InstitutionID != instID {
			if err := model.UpdateUploadSessionInstitution(r.Context(), s.db, session.ID, &instID); err == nil {
				session.InstitutionID = &instID
			}
		}
	}

	// "Ingest" — move files from staging to DICOM store directory
	study, err := s.ingestFiles(r.Context(), session, files, source)
	if err != nil {
		model.UpdateUploadSessionFailed(r.Context(), s.db, session.ID, err.Error())
		s.writeError(w, http.StatusInternalServerError, "ingest failed: "+err.Error())
		return
	}

	// Compute and store total study size (non-fatal).
	if dicomFiles, listErr := s.store.List(r.Context(), "dicom/raw/"+study.StudyInstanceUID); listErr == nil {
		if sizeBytes := sumStoredSizes(r.Context(), s.store, dicomFiles); sizeBytes > 0 {
			model.UpdateStudySizeBytes(r.Context(), s.db, study.ID, sizeBytes)
			study.StudySizeBytes = sizeBytes
		}
	}

	if session.UploaderEmail != "" {
		projectName := projectNameForStudy(r.Context(), s.db, study.ProjectID)
		subject, body := email.UploadConfirmed(study.StudyInstanceUID, study.Modality, projectName, len(files), study.CreatedAt)
		if err := s.mailer.Send(r.Context(), session.UploaderEmail, subject, body); err != nil {
			log.Printf("upload confirm email to %s: %v", session.UploaderEmail, err)
		}
	}

	// Audit
	model.CreateAuditEntry(r.Context(), s.db, "upload.complete", uploadActor(r), "study", study.ID, clientIP(r), map[string]any{
		"session_id":        session.ID,
		"study_uid":         study.StudyInstanceUID,
		"modality":          study.Modality,
		"instance_count":    len(files),
		"defacing_required": study.DefacingRequired,
	})
	// Detach cancellation: the goroutine outlives the request, and a cancelled
	// context would fail the delivery's own DB queries. Values (tenant ctx) kept.
	go webhook.Deliver(context.WithoutCancel(r.Context()), s.db, "study.created", study)

	s.writeJSON(w, http.StatusOK, uploadCompleteResponse{
		SessionID: session.ID,
		Status:    "completed",
		Study:     study,
	})
}

func (s *Server) ingestFiles(ctx context.Context, session *model.UploadSession, files []string, source string) (*model.Study, error) {
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

	// Best-effort extraction of pseudonymized PatientID + StudyDate from
	// the first uploaded .dcm. The client-side anonymizer writes the new
	// SUBJ-<hex> PatientID into tag 0010,0020 before upload, so reading
	// it here is the canonical source of truth for the subject identifier.
	// StudyDate from the file header only overrides the session metadata
	// when the client didn't provide it.
	dicomSubjectID, dicomStudyDate := s.extractIngestMetadata(ctx, fmt.Sprintf("%s/0.dcm", dstPrefix))
	studyDate := session.StudyDate
	if studyDate == nil && dicomStudyDate != "" {
		sd := dicomStudyDate
		studyDate = &sd
	}
	var subjectIDPtr *string
	if dicomSubjectID != "" {
		subjectIDPtr = &dicomSubjectID
	}

	// Create study record. subject_id is the canonical subject identifier;
	// initializing it from the DICOM-derived value at ingest puts the
	// study into the XNAT-style subject listing immediately. Researchers
	// can later re-key it to merge patients across studies.
	study := &model.Study{
		ProjectID:        session.ProjectID,
		UploadSessionID:  &session.ID,
		InstitutionID:    session.InstitutionID,
		StudyInstanceUID: studyUID,
		Modality:         deref(session.Modality),
		BodyPart:         deref(session.BodyPart),
		StudyDescription: "",
		StudyDate:        studyDate,
		SubjectID:        subjectIDPtr,
		SeriesCount:      0,
		InstanceCount:    len(files),
		Status:           "received",
		DefacingRequired: defacingRequired,
		DicomStore:       "raw",
		Source:           source,
	}
	if err := model.CreateStudy(ctx, s.db, study); err != nil {
		return nil, fmt.Errorf("create study: %w", err)
	}

	// Extract per-series DICOM metadata. Best-effort: a failure here does not
	// fail the upload. Reads the moved files (at dicom/raw/{studyUID}/*.dcm)
	// and groups them by SeriesInstanceUID, then upserts one row per series.
	if err := s.upsertSeriesMetadataForStudy(ctx, study.ID, studyUID); err != nil {
		log.Printf("upload ingest: extract series metadata for %s: %v", studyUID, err)
	}

	// Evaluate routing rules — may mutate study (e.g. auto_approve, require_defacing).
	routing.EvaluateRules(ctx, s.db, s.store, study)

	// Auto-dispatch processing pipeline (classification → defacing → QC → BIDS, etc.)
	s.AdvancePipeline(ctx, study.ID)

	// Update session
	model.UpdateUploadSessionComplete(ctx, s.db, session.ID, studyUID, study.Modality, study.BodyPart)

	return study, nil
}

// upsertSeriesMetadataForStudy walks every dicom/raw/{studyUID}/*.dcm file in
// storage, groups them by SeriesInstanceUID, and upserts one
// study_series_metadata row per series using the first dataset seen for each.
// It is safe to call repeatedly (the upsert ON CONFLICT clause handles
// re-ingest), and is best-effort — a single bad file does not abort the rest.
func (s *Server) upsertSeriesMetadataForStudy(ctx context.Context, studyID, studyUID string) error {
	keys, err := s.store.List(ctx, fmt.Sprintf("dicom/raw/%s/", studyUID))
	if err != nil {
		return fmt.Errorf("list dicom files: %w", err)
	}

	type seriesAgg struct {
		first         dicomlib.Dataset
		hasFirst      bool
		instanceCount int
	}
	bySeries := make(map[string]*seriesAgg)
	var order []string

	for _, key := range keys {
		if !strings.HasSuffix(key, ".dcm") {
			continue
		}
		seriesUID, dataset, ok := readSeriesUIDAndDataset(ctx, s.store, key)
		if !ok || seriesUID == "" {
			continue
		}
		agg, exists := bySeries[seriesUID]
		if !exists {
			agg = &seriesAgg{first: dataset, hasFirst: true}
			bySeries[seriesUID] = agg
			order = append(order, seriesUID)
		}
		agg.instanceCount++
	}

	for _, seriesUID := range order {
		agg := bySeries[seriesUID]
		if agg == nil || !agg.hasFirst {
			continue
		}
		sm := extractSeriesMetadata(agg.first)
		if sm == nil {
			continue
		}
		sm.StudyID = studyID
		if sm.SeriesInstanceUID == "" {
			sm.SeriesInstanceUID = seriesUID
		}
		sm.InstanceCount = agg.instanceCount
		if err := model.UpsertSeriesMetadata(ctx, s.db, sm); err != nil {
			log.Printf("upsert series metadata %s/%s: %v", studyUID, seriesUID, err)
		}
	}
	return nil
}

// readSeriesUIDAndDataset retrieves and parses a single DICOM file's header
// from storage. The dataset is returned without pixel data. Returns ok=false
// when the file can't be read or parsed; callers should skip it and continue.
func readSeriesUIDAndDataset(ctx context.Context, store storage.Storage, key string) (string, dicomlib.Dataset, bool) {
	rc, err := store.Retrieve(ctx, key)
	if err != nil {
		return "", dicomlib.Dataset{}, false
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	if err != nil || len(data) == 0 {
		return "", dicomlib.Dataset{}, false
	}
	dataset, err := dicomlib.Parse(
		bytes.NewReader(data), int64(len(data)), nil,
		dicomlib.SkipPixelData(),
		dicomlib.AllowMissingMetaElementGroupLength(),
	)
	if err != nil {
		return "", dicomlib.Dataset{}, false
	}
	return getStringTag(dataset, tag.SeriesInstanceUID), dataset, true
}

// uploadActor names the audit actor for upload events: the authenticated
// user's email when the request carried a session/API key, otherwise
// "anonymous" (open local-dev mode and satellite-cert uploads).
func uploadActor(r *http.Request) string {
	if u := middleware.UserFromContext(r.Context()); u != nil {
		return u.Email
	}
	return "anonymous"
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

// extractIngestMetadata best-effort reads the first stored .dcm and returns
// (subjectID, studyDate) from its DICOM header. The subjectID value is the
// pseudonymized PatientID that the client-side anonymizer wrote into tag
// 0010,0020 before upload. Errors are logged and swallowed — extraction is
// opportunistic; missing tags simply yield "" and leave the corresponding
// column NULL.
func (s *Server) extractIngestMetadata(ctx context.Context, key string) (string, string) {
	rc, err := s.store.Retrieve(ctx, key)
	if err != nil {
		log.Printf("extract ingest metadata: retrieve %s: %v", key, err)
		return "", ""
	}
	defer rc.Close()

	// 1 MB is generous — PatientID + StudyDate sit in the metadata before
	// pixel data, which starts much later in any reasonable DICOM file.
	data, err := io.ReadAll(io.LimitReader(rc, 1<<20))
	if err != nil {
		log.Printf("extract ingest metadata: read %s: %v", key, err)
		return "", ""
	}
	dataset, err := dicomlib.Parse(
		bytes.NewReader(data), int64(len(data)), nil,
		dicomlib.SkipPixelData(),
		dicomlib.AllowMissingMetaElementGroupLength(),
	)
	if err != nil {
		log.Printf("extract ingest metadata: parse %s: %v", key, err)
		return "", ""
	}
	return stowGetStringTag(dataset, tag.PatientID), stowGetStringTag(dataset, tag.StudyDate)
}
