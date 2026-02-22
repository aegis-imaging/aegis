package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/webhook"
)

// phiScanRequest is the payload sent to the Python PHI detection service.
type phiScanRequest struct {
	StudyUID   string   `json:"study_uid"`
	InputPaths []string `json:"input_paths"`
}

// phiRegion describes a single detected text region in a DICOM image.
type phiRegion struct {
	Text       string  `json:"text"`
	Confidence float64 `json:"confidence"`
	BBox       [4]int  `json:"bbox"` // [x, y, width, height]
}

// phiFinding groups detected regions for a single file.
type phiFinding struct {
	File    string      `json:"file"`
	Regions []phiRegion `json:"regions"`
}

// phiScanServiceResponse is the response from the Python PHI detection service.
type phiScanServiceResponse struct {
	StudyUID        string       `json:"study_uid"`
	Status          string       `json:"status"` // "complete" | "failed"
	PhiDetected     bool         `json:"phi_detected"`
	Findings        []phiFinding `json:"findings"`
	ToolUsed        string       `json:"tool_used"`
	DurationSeconds float64      `json:"duration_seconds"`
	Error           *string      `json:"error"`
}

// TriggerPhiScan triggers the burned-in PHI detection pipeline for a study.
// It dispatches to the Python service asynchronously and returns 202 Accepted.
func (s *Server) TriggerPhiScan(w http.ResponseWriter, r *http.Request) {
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

	if !study.PhiScanRequired {
		s.writeError(w, http.StatusBadRequest, "study does not require PHI scan")
		return
	}

	if study.PhiScanStatus == "scanning" {
		s.writeError(w, http.StatusConflict, "PHI scan already in progress")
		return
	}

	if err := model.UpdatePhiScanStatus(r.Context(), s.db, study.ID, "scanning"); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to update PHI scan status")
		return
	}

	model.CreateAuditEntry(r.Context(), s.db, "phi_scan.triggered", actorEmail(r), "study", study.ID, clientIP(r), map[string]any{
		"study_uid": studyUID,
	})

	if s.cfg.PhiDetectionServiceURL == "" {
		log.Printf("phi_scan: PHI_DETECTION_SERVICE_URL not set — study %s queued but not processed", studyUID)
		s.writeJSON(w, http.StatusAccepted, map[string]string{
			"status":  "scanning",
			"message": "Study queued for PHI scan — set PHI_DETECTION_SERVICE_URL to enable processing",
		})
		return
	}

	go s.runPhiScan(study)

	s.writeJSON(w, http.StatusAccepted, map[string]string{
		"status":  "scanning",
		"message": "PHI scan pipeline started",
	})
}

// runPhiScan calls the Python PHI detection service and updates the study record.
func (s *Server) runPhiScan(study *model.Study) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	studyUID := study.StudyInstanceUID

	// Scan the raw store — we want to detect PHI before any defacing.
	files, err := s.listDicomFiles("raw", studyUID)
	if err != nil || len(files) == 0 {
		log.Printf("phi_scan: no files found for study %s: %v", studyUID, err)
		model.UpdatePhiScanStatus(ctx, s.db, study.ID, "failed")
		return
	}

	payload := phiScanRequest{
		StudyUID:   studyUID,
		InputPaths: files,
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		s.cfg.PhiDetectionServiceURL+"/detect", bytes.NewReader(body))
	if err != nil {
		log.Printf("phi_scan: build request for %s: %v", studyUID, err)
		model.UpdatePhiScanStatus(ctx, s.db, study.ID, "failed")
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		log.Printf("phi_scan: call service for %s: %v", studyUID, err)
		model.UpdatePhiScanStatus(ctx, s.db, study.ID, "failed")
		return
	}
	defer resp.Body.Close()

	var svcResp phiScanServiceResponse
	if err := json.NewDecoder(resp.Body).Decode(&svcResp); err != nil {
		log.Printf("phi_scan: decode response for %s: %v", studyUID, err)
		model.UpdatePhiScanStatus(ctx, s.db, study.ID, "failed")
		return
	}

	if svcResp.Status != "complete" {
		errMsg := "unknown error"
		if svcResp.Error != nil {
			errMsg = *svcResp.Error
		}
		log.Printf("phi_scan: service failed for %s (%s): %s", studyUID, svcResp.ToolUsed, errMsg)
		model.UpdatePhiScanStatus(ctx, s.db, study.ID, "failed")
		model.CreateAuditEntry(ctx, s.db, "phi_scan.failed", "system", "study", study.ID, "", map[string]any{
			"study_uid": studyUID,
			"tool":      svcResp.ToolUsed,
			"error":     errMsg,
		})
		s.notifyPipelineFailure(ctx, studyUID, "phi_scan", errMsg)
		return
	}

	// Determine result status.
	newStatus := "clean"
	if svcResp.PhiDetected {
		newStatus = "flagged"
	}

	if err := model.UpdatePhiScanStatus(ctx, s.db, study.ID, newStatus); err != nil {
		log.Printf("phi_scan: update study record for %s: %v", studyUID, err)
		return
	}
	if newStatus == "flagged" {
		webhook.Deliver(ctx, s.db, "study.phi_flagged", study)
	}

	log.Printf("phi_scan: complete for %s — phi_detected=%v, %d files in %.1fs using %s",
		studyUID, svcResp.PhiDetected, len(files), svcResp.DurationSeconds, svcResp.ToolUsed)

	// Store findings in audit detail for review.
	detail := map[string]any{
		"study_uid":        studyUID,
		"tool":             svcResp.ToolUsed,
		"phi_detected":     svcResp.PhiDetected,
		"files_scanned":    len(files),
		"duration_seconds": fmt.Sprintf("%.1f", svcResp.DurationSeconds),
	}
	if svcResp.PhiDetected {
		detail["findings"] = svcResp.Findings
	}
	model.CreateAuditEntry(ctx, s.db, "phi_scan.complete", "system", "study", study.ID, "", detail)

	// Advance pipeline — may dispatch next eligible services.
	s.AdvancePipeline(ctx, study.ID)
}
