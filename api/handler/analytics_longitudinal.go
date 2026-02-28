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

// longitudinalAnalyticsBody is the request body for triggering longitudinal analytics.
type longitudinalAnalyticsBody struct {
	BaselineStudyID  string  `json:"baseline_study_id"`
	ScanIntervalDays float64 `json:"scan_interval_days"`
}

// longitudinalAnalyticsRequest is sent to the Python analytics service.
type longitudinalAnalyticsRequest struct {
	BaselineStudyUID string   `json:"baseline_study_uid"`
	FollowupStudyUID string   `json:"followup_study_uid"`
	BaselineBidsDir  string   `json:"baseline_bids_dir"`
	FollowupBidsDir  string   `json:"followup_bids_dir"`
	OutputDir        string   `json:"output_dir"`
	ScanIntervalDays float64  `json:"scan_interval_days"`
	Tools            []string `json:"tools,omitempty"`
	Atlas            string   `json:"atlas,omitempty"`
}

// TriggerLongitudinalAnalytics triggers paired longitudinal analytics (e.g. TBM-SyN)
// for a follow-up study against its baseline.
//
// POST /api/studies/{studyUID}/longitudinal-analytics
// Body: {"baseline_study_id": "<uuid>", "scan_interval_days": 365}
func (s *Server) TriggerLongitudinalAnalytics(w http.ResponseWriter, r *http.Request) {
	followupUID := r.PathValue("studyUID")
	if followupUID == "" {
		s.writeError(w, http.StatusBadRequest, "missing study UID")
		return
	}

	var body longitudinalAnalyticsBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if body.BaselineStudyID == "" {
		s.writeError(w, http.StatusBadRequest, "baseline_study_id is required")
		return
	}

	if body.ScanIntervalDays <= 0 {
		s.writeError(w, http.StatusBadRequest, "scan_interval_days must be positive")
		return
	}

	// Resolve follow-up study
	followup, err := model.GetStudyByUID(r.Context(), s.db, followupUID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "follow-up study not found")
		return
	}

	// Resolve baseline study
	baseline, err := model.GetStudyByID(r.Context(), s.db, body.BaselineStudyID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "baseline study not found")
		return
	}

	// Validate BIDS conversion complete on both
	if followup.BidsRequired && followup.BidsStatus != "complete" {
		s.writeError(w, http.StatusBadRequest, "follow-up study BIDS conversion must be complete")
		return
	}
	if baseline.BidsRequired && baseline.BidsStatus != "complete" {
		s.writeError(w, http.StatusBadRequest, "baseline study BIDS conversion must be complete")
		return
	}

	// Validate analytics not already running on follow-up
	if followup.AnalyticsStatus == "analyzing" {
		s.writeError(w, http.StatusConflict, "analytics already in progress for follow-up study")
		return
	}

	// Set follow-up analytics status to analyzing
	if err := model.UpdateAnalyticsStatus(r.Context(), s.db, followup.ID, "analyzing"); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to update analytics status")
		return
	}

	model.CreateAuditEntry(r.Context(), s.db, "analytics.longitudinal_triggered", actorEmail(r),
		"study", followup.ID, clientIP(r), map[string]any{
			"followup_study_uid": followupUID,
			"baseline_study_uid": baseline.StudyInstanceUID,
			"baseline_study_id":  baseline.ID,
			"scan_interval_days": body.ScanIntervalDays,
		})

	if s.cfg.AnalyticsServiceURL == "" {
		log.Printf("analytics: ANALYTICS_SERVICE_URL not set — longitudinal for %s queued but not processed", followupUID)
		s.writeJSON(w, http.StatusAccepted, map[string]string{
			"status":  "analyzing",
			"message": "Longitudinal analytics queued — set ANALYTICS_SERVICE_URL to enable processing",
		})
		return
	}

	go s.runLongitudinalAnalytics(followup, baseline, body.ScanIntervalDays)

	s.writeJSON(w, http.StatusAccepted, map[string]any{
		"status":             "analyzing",
		"message":            "Longitudinal analytics pipeline started (TBM-SyN)",
		"scan_interval_days": body.ScanIntervalDays,
		"baseline_study_uid": baseline.StudyInstanceUID,
		"followup_study_uid": followupUID,
	})
}

// runLongitudinalAnalytics calls the Python service and updates the follow-up study record.
func (s *Server) runLongitudinalAnalytics(followup, baseline *model.Study, scanIntervalDays float64) {
	ctx, cancel := context.WithTimeout(context.Background(), 24*time.Hour)
	defer cancel()

	followupUID := followup.StudyInstanceUID
	baselineUID := baseline.StudyInstanceUID

	baselineBidsDir := filepath.Join(s.cfg.LocalStorageDir, "bids", baselineUID)
	followupBidsDir := filepath.Join(s.cfg.LocalStorageDir, "bids", followupUID)
	outputDir := filepath.Join(s.cfg.LocalStorageDir, "analytics", followupUID)

	payload := longitudinalAnalyticsRequest{
		BaselineStudyUID: baselineUID,
		FollowupStudyUID: followupUID,
		BaselineBidsDir:  baselineBidsDir,
		FollowupBidsDir:  followupBidsDir,
		OutputDir:        outputDir,
		ScanIntervalDays: scanIntervalDays,
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		s.cfg.AnalyticsServiceURL+"/analyze-longitudinal", bytes.NewReader(body))
	if err != nil {
		log.Printf("analytics-longitudinal: build request for %s: %v", followupUID, err)
		model.UpdateAnalyticsStatus(ctx, s.db, followup.ID, "failed")
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		log.Printf("analytics-longitudinal: call service for %s: %v", followupUID, err)
		model.UpdateAnalyticsStatus(ctx, s.db, followup.ID, "failed")
		return
	}
	defer resp.Body.Close()

	var svcResp analyticsServiceResponse
	if err := json.NewDecoder(resp.Body).Decode(&svcResp); err != nil {
		log.Printf("analytics-longitudinal: decode response for %s: %v", followupUID, err)
		model.UpdateAnalyticsStatus(ctx, s.db, followup.ID, "failed")
		return
	}

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
		log.Printf("analytics-longitudinal: service failed for %s: %s", followupUID, errMsg)
		model.UpdateAnalyticsStatus(ctx, s.db, followup.ID, "failed")
		model.CreateAuditEntry(ctx, s.db, "analytics.longitudinal_failed", "system",
			"study", followup.ID, "", map[string]any{
				"followup_study_uid": followupUID,
				"baseline_study_uid": baselineUID,
				"error":              errMsg,
			})
		s.notifyPipelineFailure(ctx, followupUID, "analytics-longitudinal", errMsg)
		return
	}

	if err := model.UpdateAnalyticsStatus(ctx, s.db, followup.ID, newStatus); err != nil {
		log.Printf("analytics-longitudinal: update study record for %s: %v", followupUID, err)
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

	log.Printf("analytics-longitudinal: %s for %s — %d tools in %.1fs",
		newStatus, followupUID, len(svcResp.Results), svcResp.DurationSeconds)

	model.CreateAuditEntry(ctx, s.db, "analytics.longitudinal_complete", "system",
		"study", followup.ID, "", map[string]any{
			"followup_study_uid": followupUID,
			"baseline_study_uid": baselineUID,
			"status":             newStatus,
			"tools":              toolSummary,
			"duration_seconds":   fmt.Sprintf("%.1f", svcResp.DurationSeconds),
			"scan_interval_days": scanIntervalDays,
		})

	s.AdvancePipeline(ctx, followup.ID)
}
