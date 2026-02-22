package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/aegis-imaging/aegis/api/model"
)

// defaceRequest is the payload sent to the Python defacing service.
type defaceRequest struct {
	StudyUID   string   `json:"study_uid"`
	InputPaths []string `json:"input_paths"`
	OutputDir  string   `json:"output_dir"`
}

// defaceServiceResponse is the response from the Python defacing service.
type defaceServiceResponse struct {
	StudyUID        string   `json:"study_uid"`
	Status          string   `json:"status"` // "complete" | "failed"
	OutputPaths     []string `json:"output_paths"`
	ToolUsed        string   `json:"tool_used"`
	DurationSeconds float64  `json:"duration_seconds"`
	SsimScore       *float64 `json:"ssim_score"`
	Error           *string  `json:"error"`
}

// TriggerDeface triggers the defacing pipeline for a study.
// It dispatches to the Python defacing service asynchronously and returns
// 202 Accepted immediately. Status updates are written to the study record.
func (s *Server) TriggerDeface(w http.ResponseWriter, r *http.Request) {
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

	if !study.DefacingRequired {
		s.writeError(w, http.StatusBadRequest, "study does not require defacing")
		return
	}

	if study.Status == "defacing" {
		s.writeError(w, http.StatusConflict, "defacing already in progress")
		return
	}

	if err := model.UpdateStudyStatus(r.Context(), s.db, study.ID, "defacing"); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to update study status")
		return
	}

	model.CreateAuditEntry(r.Context(), s.db, "deface.triggered", actorEmail(r), "study", study.ID, clientIP(r), map[string]any{
		"study_uid": studyUID,
	})

	// If no defacing service is configured, leave in "defacing" state (will be processed later).
	if s.cfg.DefacingServiceURL == "" {
		log.Printf("deface: DEFACING_SERVICE_URL not set — study %s queued but not processed", studyUID)
		s.writeJSON(w, http.StatusAccepted, map[string]string{
			"status":  "defacing",
			"message": "Study queued for defacing — set DEFACING_SERVICE_URL to enable processing",
		})
		return
	}

	// Dispatch asynchronously so the caller doesn't wait minutes for defacing to finish.
	go s.runDefacing(study)

	s.writeJSON(w, http.StatusAccepted, map[string]string{
		"status":  "defacing",
		"message": "Defacing pipeline started",
	})
}

// runDefacing calls the Python defacing service and updates the study record.
// Runs in a background goroutine; logs errors but does not panic.
func (s *Server) runDefacing(study *model.Study) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	studyUID := study.StudyInstanceUID

	// List raw DICOM files for this study.
	files, err := s.listDicomFiles("raw", studyUID)
	if err != nil || len(files) == 0 {
		log.Printf("deface: no files found for study %s: %v", studyUID, err)
		model.UpdateStudyStatus(ctx, s.db, study.ID, "received")
		return
	}

	// Output directory for defaced files in the clean store.
	cleanDir := filepath.Join(s.cfg.LocalStorageDir, "dicom", "clean", studyUID)
	if err := os.MkdirAll(cleanDir, 0755); err != nil {
		log.Printf("deface: cannot create clean dir for %s: %v", studyUID, err)
		model.UpdateStudyStatus(ctx, s.db, study.ID, "received")
		return
	}

	payload := defaceRequest{
		StudyUID:   studyUID,
		InputPaths: files,
		OutputDir:  cleanDir,
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		s.cfg.DefacingServiceURL+"/deface", bytes.NewReader(body))
	if err != nil {
		log.Printf("deface: build request for %s: %v", studyUID, err)
		model.UpdateStudyStatus(ctx, s.db, study.ID, "received")
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		log.Printf("deface: call service for %s: %v", studyUID, err)
		model.UpdateStudyStatus(ctx, s.db, study.ID, "received")
		return
	}
	defer resp.Body.Close()

	var svcResp defaceServiceResponse
	if err := json.NewDecoder(resp.Body).Decode(&svcResp); err != nil {
		log.Printf("deface: decode response for %s: %v", studyUID, err)
		model.UpdateStudyStatus(ctx, s.db, study.ID, "received")
		return
	}

	if svcResp.Status != "complete" {
		errMsg := "unknown error"
		if svcResp.Error != nil {
			errMsg = *svcResp.Error
		}
		log.Printf("deface: service failed for %s (%s): %s", studyUID, svcResp.ToolUsed, errMsg)
		model.UpdateStudyStatus(ctx, s.db, study.ID, "received")
		model.CreateAuditEntry(ctx, s.db, "deface.failed", "system", "study", study.ID, "", map[string]any{
			"study_uid": studyUID,
			"tool":      svcResp.ToolUsed,
			"error":     errMsg,
		})
		s.notifyPipelineFailure(ctx, studyUID, "defacing", errMsg)
		return
	}

	if err := model.UpdateStudyDefaced(ctx, s.db, study.ID); err != nil {
		log.Printf("deface: update study record for %s: %v", studyUID, err)
		return
	}

	if svcResp.SsimScore != nil {
		if err := model.UpdateDefaceQaScore(ctx, s.db, study.ID, *svcResp.SsimScore); err != nil {
			log.Printf("deface: store qa score for %s: %v", studyUID, err)
		}
	}

	log.Printf("deface: complete for %s — %d files in %.1fs using %s",
		studyUID, len(svcResp.OutputPaths), svcResp.DurationSeconds, svcResp.ToolUsed)

	auditDetails := map[string]any{
		"study_uid":        studyUID,
		"tool":             svcResp.ToolUsed,
		"output_files":     len(svcResp.OutputPaths),
		"duration_seconds": fmt.Sprintf("%.1f", svcResp.DurationSeconds),
	}
	if svcResp.SsimScore != nil {
		auditDetails["ssim_score"] = fmt.Sprintf("%.4f", *svcResp.SsimScore)
	}
	model.CreateAuditEntry(ctx, s.db, "deface.complete", "system", "study", study.ID, "", auditDetails)

	// Advance pipeline — unblocks Phase 2 (QC, BIDS) which wait for defacing.
	s.AdvancePipeline(ctx, study.ID)
}
