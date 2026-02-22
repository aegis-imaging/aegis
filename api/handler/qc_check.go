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
)

// qcCheckRequest is the payload sent to the Python QC service.
type qcCheckRequest struct {
	StudyUID   string   `json:"study_uid"`
	InputPaths []string `json:"input_paths"`
}

// qcCheckDetail describes a single QC check result.
type qcCheckDetail struct {
	Name    string         `json:"name"`
	Status  string         `json:"status"`
	Message string         `json:"message"`
	Details map[string]any `json:"details"`
}

// qcCheckServiceResponse is the response from the Python QC service.
type qcCheckServiceResponse struct {
	StudyUID        string          `json:"study_uid"`
	Status          string          `json:"status"` // "complete" | "failed"
	OverallQuality  string          `json:"overall_quality"`
	QualityIssues   []qcCheckDetail `json:"quality_issues"`
	ToolUsed        string          `json:"tool_used"`
	DurationSeconds float64         `json:"duration_seconds"`
	Error           *string         `json:"error"`
}

// TriggerQcCheck triggers the QC automation pipeline for a study.
// It dispatches to the Python service asynchronously and returns 202 Accepted.
func (s *Server) TriggerQcCheck(w http.ResponseWriter, r *http.Request) {
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

	if !study.QcRequired {
		s.writeError(w, http.StatusBadRequest, "study does not require QC check")
		return
	}

	if study.QcStatus == "checking" {
		s.writeError(w, http.StatusConflict, "QC check already in progress")
		return
	}

	if err := model.UpdateQcStatus(r.Context(), s.db, study.ID, "checking"); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to update QC status")
		return
	}

	model.CreateAuditEntry(r.Context(), s.db, "qc_check.triggered", actorEmail(r), "study", study.ID, clientIP(r), map[string]any{
		"study_uid": studyUID,
	})

	if s.cfg.QcServiceURL == "" {
		log.Printf("qc_check: QC_SERVICE_URL not set — study %s queued but not processed", studyUID)
		s.writeJSON(w, http.StatusAccepted, map[string]string{
			"status":  "checking",
			"message": "Study queued for QC check — set QC_SERVICE_URL to enable processing",
		})
		return
	}

	go s.runQcCheck(study)

	s.writeJSON(w, http.StatusAccepted, map[string]string{
		"status":  "checking",
		"message": "QC check pipeline started",
	})
}

// runQcCheck calls the Python QC service and updates the study record.
func (s *Server) runQcCheck(study *model.Study) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	studyUID := study.StudyInstanceUID

	files, err := s.listDicomFiles(study.DicomStore, studyUID)
	if err != nil || len(files) == 0 {
		log.Printf("qc_check: no files found for study %s: %v", studyUID, err)
		model.UpdateQcStatus(ctx, s.db, study.ID, "failed")
		return
	}

	payload := qcCheckRequest{
		StudyUID:   studyUID,
		InputPaths: files,
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		s.cfg.QcServiceURL+"/check", bytes.NewReader(body))
	if err != nil {
		log.Printf("qc_check: build request for %s: %v", studyUID, err)
		model.UpdateQcStatus(ctx, s.db, study.ID, "failed")
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		log.Printf("qc_check: call service for %s: %v", studyUID, err)
		model.UpdateQcStatus(ctx, s.db, study.ID, "failed")
		return
	}
	defer resp.Body.Close()

	var svcResp qcCheckServiceResponse
	if err := json.NewDecoder(resp.Body).Decode(&svcResp); err != nil {
		log.Printf("qc_check: decode response for %s: %v", studyUID, err)
		model.UpdateQcStatus(ctx, s.db, study.ID, "failed")
		return
	}

	if svcResp.Status != "complete" {
		errMsg := "unknown error"
		if svcResp.Error != nil {
			errMsg = *svcResp.Error
		}
		log.Printf("qc_check: service failed for %s (%s): %s", studyUID, svcResp.ToolUsed, errMsg)
		model.UpdateQcStatus(ctx, s.db, study.ID, "failed")
		model.CreateAuditEntry(ctx, s.db, "qc_check.failed", "system", "study", study.ID, "", map[string]any{
			"study_uid": studyUID,
			"tool":      svcResp.ToolUsed,
			"error":     errMsg,
		})
		s.notifyPipelineFailure(ctx, studyUID, "qc_check", errMsg)
		return
	}

	newStatus := svcResp.OverallQuality
	if err := model.UpdateQcStatus(ctx, s.db, study.ID, newStatus); err != nil {
		log.Printf("qc_check: update study record for %s: %v", studyUID, err)
		return
	}

	log.Printf("qc_check: complete for %s — overall=%s, %d checks in %.1fs using %s",
		studyUID, svcResp.OverallQuality, len(svcResp.QualityIssues), svcResp.DurationSeconds, svcResp.ToolUsed)

	model.CreateAuditEntry(ctx, s.db, "qc_check.complete", "system", "study", study.ID, "", map[string]any{
		"study_uid":        studyUID,
		"tool":             svcResp.ToolUsed,
		"overall_quality":  svcResp.OverallQuality,
		"files_checked":    len(files),
		"duration_seconds": fmt.Sprintf("%.1f", svcResp.DurationSeconds),
		"quality_issues":   svcResp.QualityIssues,
	})

	// Advance pipeline — may dispatch next eligible services.
	s.AdvancePipeline(ctx, study.ID)
}
