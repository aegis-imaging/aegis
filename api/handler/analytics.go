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

// analyticsRequest is the payload sent to the Python analytics service.
type analyticsRequest struct {
	StudyUID  string   `json:"study_uid"`
	BidsDir   string   `json:"bids_dir"`
	OutputDir string   `json:"output_dir"`
	Tools     []string `json:"tools,omitempty"`
}

// analyticsToolResult is a per-tool result from the analytics service.
type analyticsToolResult struct {
	Tool            string         `json:"tool"`
	Success         bool           `json:"success"`
	DurationSeconds float64        `json:"duration_seconds"`
	Outputs         []string       `json:"outputs"`
	Metrics         map[string]any `json:"metrics"`
	Error           string         `json:"error"`
}

// analyticsServiceResponse is the response from the Python analytics service.
type analyticsServiceResponse struct {
	StudyUID        string                `json:"study_uid"`
	Status          string                `json:"status"` // "complete" | "partial" | "failed"
	Results         []analyticsToolResult `json:"results"`
	DurationSeconds float64               `json:"duration_seconds"`
	Error           *string               `json:"error"`
}

// triggerAnalyticsBody is the optional request body for TriggerAnalytics.
type triggerAnalyticsBody struct {
	// Tool specifies which analytics backend to run. Empty = auto-select.
	Tool string `json:"tool"`
}

// TriggerAnalytics triggers the neuroimaging analytics pipeline for a study.
// Analytics runs after BIDS conversion is complete, consuming NIfTI outputs.
// An optional JSON body {"tool": "<backend>"} selects a specific backend.
// Manual triggers auto-enable analytics_required on studies that don't have it set.
func (s *Server) TriggerAnalytics(w http.ResponseWriter, r *http.Request) {
	studyUID := r.PathValue("studyUID")
	if studyUID == "" {
		s.writeError(w, http.StatusBadRequest, "missing study UID")
		return
	}

	// Parse optional tool selection from request body.
	var body triggerAnalyticsBody
	if r.ContentLength > 0 {
		json.NewDecoder(r.Body).Decode(&body) // tolerate decode errors — tool defaults to ""
	}

	study, err := model.GetStudyByUID(r.Context(), s.db, studyUID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "study not found")
		return
	}

	if study.AnalyticsStatus == "analyzing" {
		s.writeError(w, http.StatusConflict, "analytics already in progress")
		return
	}

	// Auto-enable analytics_required when manually triggered.
	if !study.AnalyticsRequired {
		if err := model.SetAnalyticsRequired(r.Context(), s.db, study.ID, true); err != nil {
			s.writeError(w, http.StatusInternalServerError, "failed to enable analytics for study")
			return
		}
		study.AnalyticsRequired = true
	}

	// Warn (but don't block) when BIDS conversion is pending — some tools may still run.
	if study.BidsRequired && study.BidsStatus != "complete" {
		log.Printf("analytics: study %s has bids_required=true but bids_status=%q — proceeding with manual trigger", studyUID, study.BidsStatus)
	}

	if err := model.UpdateAnalyticsStatus(r.Context(), s.db, study.ID, "analyzing"); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to update analytics status")
		return
	}

	auditMeta := map[string]any{"study_uid": studyUID}
	if body.Tool != "" {
		auditMeta["tool"] = body.Tool
	}
	model.CreateAuditEntry(r.Context(), s.db, "analytics.triggered", actorEmail(r), "study", study.ID, clientIP(r), auditMeta)

	if s.cfg.AnalyticsServiceURL == "" {
		log.Printf("analytics: ANALYTICS_SERVICE_URL not set — study %s queued but not processed", studyUID)
		s.writeJSON(w, http.StatusAccepted, map[string]string{
			"status":  "analyzing",
			"message": "Study queued for analytics — set ANALYTICS_SERVICE_URL to enable processing",
		})
		return
	}

	go s.runAnalytics(study, body.Tool)

	s.writeJSON(w, http.StatusAccepted, map[string]string{
		"status":  "analyzing",
		"message": "Neuroimaging analytics pipeline started",
	})
}

// runAnalytics calls the Python analytics service and updates the study record.
// tool specifies which backend to run; empty string means auto-select.
func (s *Server) runAnalytics(study *model.Study, tool string) {
	ctx, cancel := context.WithTimeout(context.Background(), 24*time.Hour)
	defer cancel()

	studyUID := study.StudyInstanceUID

	// Input: BIDS directory for this study
	bidsDir := filepath.Join(s.cfg.LocalStorageDir, "bids", studyUID)

	// Output: analytics/{studyUID}/
	outputDir := filepath.Join(s.cfg.LocalStorageDir, "analytics", studyUID)

	payload := analyticsRequest{
		StudyUID:  studyUID,
		BidsDir:   bidsDir,
		OutputDir: outputDir,
	}
	if tool != "" {
		payload.Tools = []string{tool}
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		s.cfg.AnalyticsServiceURL+"/analyze", bytes.NewReader(body))
	if err != nil {
		log.Printf("analytics: build request for %s: %v", studyUID, err)
		model.UpdateAnalyticsStatus(ctx, s.db, study.ID, "failed")
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		log.Printf("analytics: call service for %s: %v", studyUID, err)
		model.UpdateAnalyticsStatus(ctx, s.db, study.ID, "failed")
		return
	}
	defer resp.Body.Close()

	var svcResp analyticsServiceResponse
	if err := json.NewDecoder(resp.Body).Decode(&svcResp); err != nil {
		log.Printf("analytics: decode response for %s: %v", studyUID, err)
		model.UpdateAnalyticsStatus(ctx, s.db, study.ID, "failed")
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
		log.Printf("analytics: service failed for %s: %s", studyUID, errMsg)
		model.UpdateAnalyticsStatus(ctx, s.db, study.ID, "failed")
		model.CreateAuditEntry(ctx, s.db, "analytics.failed", "system", "study", study.ID, "", map[string]any{
			"study_uid": studyUID,
			"error":     errMsg,
		})
		s.notifyPipelineFailure(ctx, studyUID, "analytics", errMsg)
		return
	}

	if err := model.UpdateAnalyticsStatus(ctx, s.db, study.ID, newStatus); err != nil {
		log.Printf("analytics: update study record for %s: %v", studyUID, err)
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

	log.Printf("analytics: %s for %s — %d tools in %.1fs",
		newStatus, studyUID, len(svcResp.Results), svcResp.DurationSeconds)

	model.CreateAuditEntry(ctx, s.db, "analytics.complete", "system", "study", study.ID, "", map[string]any{
		"study_uid":        studyUID,
		"status":           newStatus,
		"tools":            toolSummary,
		"duration_seconds": fmt.Sprintf("%.1f", svcResp.DurationSeconds),
	})

	// Store structured ROI results in the biomarker database.
	s.storeROIResults(ctx, study, svcResp.Results)

	// Advance pipeline — may dispatch next eligible services.
	s.AdvancePipeline(ctx, study.ID)
}
