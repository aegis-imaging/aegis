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

// sctRequest is the payload sent to the Python SCT service.
type sctRequest struct {
	StudyUID  string `json:"study_uid"`
	BidsDir   string `json:"bids_dir"`
	OutputDir string `json:"output_dir"`
}

// sctToolResult is a per-tool result from the SCT service.
type sctToolResult struct {
	Tool            string         `json:"tool"`
	Success         bool           `json:"success"`
	DurationSeconds float64        `json:"duration_seconds"`
	Outputs         []string       `json:"outputs"`
	Metrics         map[string]any `json:"metrics"`
	Error           string         `json:"error"`
}

// sctServiceResponse is the response from the Python SCT service.
type sctServiceResponse struct {
	StudyUID        string          `json:"study_uid"`
	Status          string          `json:"status"` // "complete" | "partial" | "failed"
	Results         []sctToolResult `json:"results"`
	DurationSeconds float64         `json:"duration_seconds"`
	Error           *string         `json:"error"`
}

// TriggerSct triggers the Spinal Cord Toolbox analysis pipeline for a study.
// SCT runs after BIDS conversion is complete, consuming NIfTI outputs.
func (s *Server) TriggerSct(w http.ResponseWriter, r *http.Request) {
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

	if !study.SctRequired {
		s.writeError(w, http.StatusBadRequest, "study does not require SCT analysis")
		return
	}

	if study.SctStatus == "analyzing" {
		s.writeError(w, http.StatusConflict, "SCT analysis already in progress")
		return
	}

	// SCT requires BIDS conversion to be complete.
	if study.BidsRequired && study.BidsStatus != "complete" {
		s.writeError(w, http.StatusBadRequest, "BIDS conversion must complete before SCT can run")
		return
	}

	if err := model.UpdateSctStatus(r.Context(), s.db, study.ID, "analyzing"); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to update SCT status")
		return
	}

	model.CreateAuditEntry(r.Context(), s.db, "sct.triggered", actorEmail(r), "study", study.ID, clientIP(r), map[string]any{
		"study_uid": studyUID,
	})

	if s.cfg.SctServiceURL == "" {
		log.Printf("sct: SCT_SERVICE_URL not set — study %s queued but not processed", studyUID)
		s.writeJSON(w, http.StatusAccepted, map[string]string{
			"status":  "analyzing",
			"message": "Study queued for SCT analysis — set SCT_SERVICE_URL to enable processing",
		})
		return
	}

	go s.runSct(study)

	s.writeJSON(w, http.StatusAccepted, map[string]string{
		"status":  "analyzing",
		"message": "Spinal Cord Toolbox analysis started",
	})
}

// runSct calls the Python SCT service and updates the study record.
func (s *Server) runSct(study *model.Study) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
	defer cancel()

	studyUID := study.StudyInstanceUID

	// Input: BIDS directory for this study
	bidsDir := filepath.Join(s.cfg.LocalStorageDir, "bids", studyUID)

	// Output: sct/{studyUID}/
	outputDir := filepath.Join(s.cfg.LocalStorageDir, "sct", studyUID)

	payload := sctRequest{
		StudyUID:  studyUID,
		BidsDir:   bidsDir,
		OutputDir: outputDir,
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		s.cfg.SctServiceURL+"/analyze", bytes.NewReader(body))
	if err != nil {
		log.Printf("sct: build request for %s: %v", studyUID, err)
		model.UpdateSctStatus(ctx, s.db, study.ID, "failed")
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		log.Printf("sct: call service for %s: %v", studyUID, err)
		model.UpdateSctStatus(ctx, s.db, study.ID, "failed")
		return
	}
	defer resp.Body.Close()

	var svcResp sctServiceResponse
	if err := json.NewDecoder(resp.Body).Decode(&svcResp); err != nil {
		log.Printf("sct: decode response for %s: %v", studyUID, err)
		model.UpdateSctStatus(ctx, s.db, study.ID, "failed")
		return
	}

	// Map service status to study status
	var newStatus string
	switch svcResp.Status {
	case "complete":
		newStatus = "complete"
	case "partial":
		newStatus = "partial"
	default:
		newStatus = "failed"
	}

	if newStatus == "failed" {
		errMsg := "unknown error"
		if svcResp.Error != nil {
			errMsg = *svcResp.Error
		}
		log.Printf("sct: service failed for %s: %s", studyUID, errMsg)
		model.UpdateSctStatus(ctx, s.db, study.ID, "failed")
		model.CreateAuditEntry(ctx, s.db, "sct.failed", "system", "study", study.ID, "", map[string]any{
			"study_uid": studyUID,
			"error":     errMsg,
		})
		s.notifyPipelineFailure(ctx, studyUID, "sct", errMsg)
		return
	}

	if err := model.UpdateSctStatus(ctx, s.db, study.ID, newStatus); err != nil {
		log.Printf("sct: update study record for %s: %v", studyUID, err)
		return
	}

	// Build per-tool summary for audit
	toolSummary := make([]map[string]any, 0, len(svcResp.Results))
	for _, r := range svcResp.Results {
		entry := map[string]any{
			"tool":     r.Tool,
			"success":  r.Success,
			"duration": fmt.Sprintf("%.1f", r.DurationSeconds),
			"outputs":  len(r.Outputs),
		}
		if r.Error != "" {
			entry["error"] = r.Error
		}
		toolSummary = append(toolSummary, entry)
	}

	log.Printf("sct: %s for %s — %d tools in %.1fs",
		newStatus, studyUID, len(svcResp.Results), svcResp.DurationSeconds)

	model.CreateAuditEntry(ctx, s.db, "sct.complete", "system", "study", study.ID, "", map[string]any{
		"study_uid":        studyUID,
		"status":           newStatus,
		"tools":            toolSummary,
		"duration_seconds": fmt.Sprintf("%.1f", svcResp.DurationSeconds),
	})

	// Store structured ROI results in the biomarker database.
	s.storeROIResults(ctx, study, convertSctToAnalyticsResults(svcResp.Results))

	// Advance pipeline — may dispatch next eligible services.
	s.AdvancePipeline(ctx, study.ID)
}

// convertSctToAnalyticsResults converts SCT tool results to analytics tool results
// so they can be stored in the biomarker database using the existing storeROIResults.
func convertSctToAnalyticsResults(results []sctToolResult) []analyticsToolResult {
	out := make([]analyticsToolResult, len(results))
	for i, r := range results {
		out[i] = analyticsToolResult{
			Tool:            r.Tool,
			Success:         r.Success,
			DurationSeconds: r.DurationSeconds,
			Outputs:         r.Outputs,
			Metrics:         r.Metrics,
			Error:           r.Error,
		}
	}
	return out
}
