package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"github.com/aegis-imaging/aegis/api/model"
)

type pixelRedactionRequest struct {
	StudyUID   string   `json:"study_uid"`
	InputPaths []string `json:"input_paths"`
	OutputDir  string   `json:"output_dir"`
}

type pixelRedactionServiceResponse struct {
	StudyUID        string       `json:"study_uid"`
	Status          string       `json:"status"`
	FilesProcessed  int          `json:"files_processed"`
	FilesRedacted   int          `json:"files_redacted"`
	Findings        []phiFinding `json:"findings"`
	ToolUsed        string       `json:"tool_used"`
	DurationSeconds float64      `json:"duration_seconds"`
	Error           *string      `json:"error"`
}

// TriggerPixelRedaction triggers the pixel-level PHI redaction pipeline for a study.
// It dispatches to the Python PHI detection service's /redact endpoint asynchronously
// and returns 202 Accepted.
func (s *Server) TriggerPixelRedaction(w http.ResponseWriter, r *http.Request) {
	studyUID := r.PathValue("studyUID")
	if studyUID == "" {
		s.writeError(w, http.StatusBadRequest, "missing study UID")
		return
	}

	study, err := model.GetStudyByUID(r.Context(), s.db, studyUID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "study not found")
		return
	}

	if !study.PixelRedactionRequired {
		s.writeError(w, http.StatusBadRequest, "study does not require pixel redaction")
		return
	}

	if study.PixelRedactionStatus == "redacting" {
		s.writeError(w, http.StatusConflict, "pixel redaction already in progress")
		return
	}

	if err := model.UpdatePixelRedactionStatus(r.Context(), s.db, study.ID, "redacting"); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to update pixel redaction status")
		return
	}

	model.CreateAuditEntry(r.Context(), s.db, "pixel_redaction.triggered", actorEmail(r), "study", study.ID, clientIP(r), map[string]any{
		"study_uid": studyUID,
	})

	if s.cfg.PhiDetectionServiceURL == "" {
		log.Printf("pixel_redaction: PHI_DETECTION_SERVICE_URL not set — study %s queued but not processed", studyUID)
		s.writeJSON(w, http.StatusAccepted, map[string]string{
			"status":  "redacting",
			"message": "Study queued for pixel redaction — set PHI_DETECTION_SERVICE_URL to enable processing",
		})
		return
	}

	go s.runPixelRedaction(study)

	s.writeJSON(w, http.StatusAccepted, map[string]string{
		"status":  "redacting",
		"message": "Pixel redaction pipeline started",
	})
}

// runPixelRedaction calls the Python PHI detection service's /redact endpoint
// and updates the study record.
func (s *Server) runPixelRedaction(study *model.Study) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	studyUID := study.StudyInstanceUID

	files, err := s.listDicomFiles("raw", studyUID)
	if err != nil || len(files) == 0 {
		log.Printf("pixel_redaction: no files found for study %s: %v", studyUID, err)
		model.UpdatePixelRedactionStatus(ctx, s.db, study.ID, "failed")
		return
	}

	// Output directory is the same raw store (in-place redaction before defacing).
	outputDir := filepath.Join(s.cfg.LocalStorageDir, "dicom", "raw", studyUID)

	payload := pixelRedactionRequest{
		StudyUID:   studyUID,
		InputPaths: files,
		OutputDir:  outputDir,
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		s.cfg.PhiDetectionServiceURL+"/redact", bytes.NewReader(body))
	if err != nil {
		log.Printf("pixel_redaction: build request for %s: %v", studyUID, err)
		model.UpdatePixelRedactionStatus(ctx, s.db, study.ID, "failed")
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		log.Printf("pixel_redaction: call service for %s: %v", studyUID, err)
		model.UpdatePixelRedactionStatus(ctx, s.db, study.ID, "failed")
		return
	}
	defer resp.Body.Close()

	var svcResp pixelRedactionServiceResponse
	if err := json.NewDecoder(resp.Body).Decode(&svcResp); err != nil {
		log.Printf("pixel_redaction: decode response for %s: %v", studyUID, err)
		model.UpdatePixelRedactionStatus(ctx, s.db, study.ID, "failed")
		return
	}

	if svcResp.Status != "complete" {
		errMsg := "unknown error"
		if svcResp.Error != nil {
			errMsg = *svcResp.Error
		}
		log.Printf("pixel_redaction: service failed for %s (%s): %s", studyUID, svcResp.ToolUsed, errMsg)
		model.UpdatePixelRedactionStatus(ctx, s.db, study.ID, "failed")
		model.CreateAuditEntry(ctx, s.db, "pixel_redaction.failed", "system", "study", study.ID, "", map[string]any{
			"study_uid": studyUID,
			"tool":      svcResp.ToolUsed,
			"error":     errMsg,
		})
		s.notifyPipelineFailure(ctx, studyUID, "pixel_redaction", errMsg)
		return
	}

	if err := model.UpdatePixelRedactionStatus(ctx, s.db, study.ID, "complete"); err != nil {
		log.Printf("pixel_redaction: update study record for %s: %v", studyUID, err)
		return
	}

	log.Printf("pixel_redaction: complete for %s — %d/%d files redacted in %.1fs using %s",
		studyUID, svcResp.FilesRedacted, svcResp.FilesProcessed, svcResp.DurationSeconds, svcResp.ToolUsed)

	detail := map[string]any{
		"study_uid":        studyUID,
		"tool":             svcResp.ToolUsed,
		"files_processed":  svcResp.FilesProcessed,
		"files_redacted":   svcResp.FilesRedacted,
		"duration_seconds": fmt.Sprintf("%.1f", svcResp.DurationSeconds),
	}
	if len(svcResp.Findings) > 0 {
		detail["findings"] = svcResp.Findings
	}
	model.CreateAuditEntry(ctx, s.db, "pixel_redaction.complete", "system", "study", study.ID, "", detail)

	s.AdvancePipeline(ctx, study.ID)
}
