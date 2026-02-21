package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/msenjem/aegis/api/model"
)

// protocolCheckRequest is the payload sent to the Python protocol service.
type protocolCheckRequest struct {
	StudyUID   string            `json:"study_uid"`
	InputPaths []string          `json:"input_paths"`
	Rules      []json.RawMessage `json:"rules"`
}

// protocolDeviceInfo describes the scanner that produced the study.
type protocolDeviceInfo struct {
	Manufacturer    string `json:"manufacturer"`
	Model           string `json:"model"`
	SoftwareVersion string `json:"software_version"`
}

// protocolFinding describes a single parameter compliance result.
type protocolFinding struct {
	TagKeyword   string   `json:"tag_keyword"`
	Severity     string   `json:"severity"`
	Status       string   `json:"status"`
	Message      string   `json:"message"`
	Expected     string   `json:"expected"`
	Actual       string   `json:"actual"`
	DeviationPct *float64 `json:"deviation_pct"`
}

// protocolCheckServiceResponse is the response from the Python protocol service.
type protocolCheckServiceResponse struct {
	StudyUID          string              `json:"study_uid"`
	Status            string              `json:"status"` // "complete" | "failed"
	OverallCompliance string              `json:"overall_compliance"`
	DeviceInfo        protocolDeviceInfo  `json:"device_info"`
	DicomFormat       string              `json:"dicom_format"`
	Findings          []protocolFinding   `json:"findings"`
	ToolUsed          string              `json:"tool_used"`
	DurationSeconds   float64             `json:"duration_seconds"`
	Error             *string             `json:"error"`
}

// TriggerProtocolCheck triggers the MRI protocol compliance pipeline for a study.
// POST /api/studies/{studyUID}/protocol-check
func (s *Server) TriggerProtocolCheck(w http.ResponseWriter, r *http.Request) {
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

	if !study.ProtocolRequired {
		s.writeError(w, http.StatusBadRequest, "study does not require protocol check")
		return
	}

	if study.ProtocolStatus == "checking" {
		s.writeError(w, http.StatusConflict, "protocol check already in progress")
		return
	}

	// Load protocol templates for this study's project.
	templates, err := model.ListProtocolTemplatesByProject(r.Context(), s.db, study.ProjectID)
	if err != nil {
		log.Printf("protocol_check: load templates for project %s: %v", study.ProjectID, err)
		s.writeError(w, http.StatusInternalServerError, "failed to load protocol templates")
		return
	}

	if len(templates) == 0 {
		s.writeError(w, http.StatusBadRequest, "no protocol templates configured for this project")
		return
	}

	if err := model.UpdateProtocolStatus(r.Context(), s.db, study.ID, "checking"); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to update protocol status")
		return
	}

	model.CreateAuditEntry(r.Context(), s.db, "protocol_check.triggered", actorEmail(r), "study", study.ID, clientIP(r), map[string]any{
		"study_uid":      studyUID,
		"template_count": len(templates),
	})

	if s.cfg.ProtocolServiceURL == "" {
		log.Printf("protocol_check: PROTOCOL_SERVICE_URL not set — study %s queued but not processed", studyUID)
		s.writeJSON(w, http.StatusAccepted, map[string]string{
			"status":  "checking",
			"message": "Study queued for protocol check — set PROTOCOL_SERVICE_URL to enable processing",
		})
		return
	}

	go s.runProtocolCheck(study, templates)

	s.writeJSON(w, http.StatusAccepted, map[string]string{
		"status":  "checking",
		"message": "Protocol compliance check started",
	})
}

// runProtocolCheck calls the Python protocol service and updates the study record.
// All enabled templates for the project are sent; the service extracts device info
// from DICOM headers and compares against each template's rules.
func (s *Server) runProtocolCheck(study *model.Study, templates []model.ProtocolTemplate) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	studyUID := study.StudyInstanceUID

	files, err := s.listDicomFiles("raw", studyUID)
	if err != nil || len(files) == 0 {
		log.Printf("protocol_check: no files found for study %s: %v", studyUID, err)
		model.UpdateProtocolStatus(ctx, s.db, study.ID, "failed")
		return
	}

	// Aggregate rules from all enabled templates.
	var allRules []json.RawMessage
	for _, t := range templates {
		if !t.Enabled {
			continue
		}
		var rules []json.RawMessage
		if err := json.Unmarshal(t.Rules, &rules); err != nil {
			log.Printf("protocol_check: parse template %s rules: %v", t.ID, err)
			continue
		}
		allRules = append(allRules, rules...)
	}

	if len(allRules) == 0 {
		log.Printf("protocol_check: no rules from templates for study %s", studyUID)
		model.UpdateProtocolStatus(ctx, s.db, study.ID, "failed")
		model.CreateAuditEntry(ctx, s.db, "protocol_check.failed", "system", "study", study.ID, "", map[string]any{
			"study_uid": studyUID,
			"error":     "no rules found in templates",
		})
		return
	}

	payload := protocolCheckRequest{
		StudyUID:   studyUID,
		InputPaths: files,
		Rules:      allRules,
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		s.cfg.ProtocolServiceURL+"/check", bytes.NewReader(body))
	if err != nil {
		log.Printf("protocol_check: build request for %s: %v", studyUID, err)
		model.UpdateProtocolStatus(ctx, s.db, study.ID, "failed")
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		log.Printf("protocol_check: call service for %s: %v", studyUID, err)
		model.UpdateProtocolStatus(ctx, s.db, study.ID, "failed")
		return
	}
	defer resp.Body.Close()

	var svcResp protocolCheckServiceResponse
	if err := json.NewDecoder(resp.Body).Decode(&svcResp); err != nil {
		log.Printf("protocol_check: decode response for %s: %v", studyUID, err)
		model.UpdateProtocolStatus(ctx, s.db, study.ID, "failed")
		return
	}

	if svcResp.Status != "complete" {
		errMsg := "unknown error"
		if svcResp.Error != nil {
			errMsg = *svcResp.Error
		}
		log.Printf("protocol_check: service failed for %s (%s): %s", studyUID, svcResp.ToolUsed, errMsg)
		model.UpdateProtocolStatus(ctx, s.db, study.ID, "failed")
		model.CreateAuditEntry(ctx, s.db, "protocol_check.failed", "system", "study", study.ID, "", map[string]any{
			"study_uid": studyUID,
			"tool":      svcResp.ToolUsed,
			"error":     errMsg,
		})
		return
	}

	newStatus := svcResp.OverallCompliance
	if err := model.UpdateProtocolStatus(ctx, s.db, study.ID, newStatus); err != nil {
		log.Printf("protocol_check: update study record for %s: %v", studyUID, err)
		return
	}

	log.Printf("protocol_check: complete for %s — overall=%s, %d findings in %.1fs using %s (format=%s, device=%s %s %s)",
		studyUID, svcResp.OverallCompliance, len(svcResp.Findings), svcResp.DurationSeconds,
		svcResp.ToolUsed, svcResp.DicomFormat,
		svcResp.DeviceInfo.Manufacturer, svcResp.DeviceInfo.Model, svcResp.DeviceInfo.SoftwareVersion)

	model.CreateAuditEntry(ctx, s.db, "protocol_check.complete", "system", "study", study.ID, "", map[string]any{
		"study_uid":          studyUID,
		"tool":               svcResp.ToolUsed,
		"overall_compliance": svcResp.OverallCompliance,
		"dicom_format":       svcResp.DicomFormat,
		"device_manufacturer": svcResp.DeviceInfo.Manufacturer,
		"device_model":        svcResp.DeviceInfo.Model,
		"device_software":     svcResp.DeviceInfo.SoftwareVersion,
		"files_checked":       len(files),
		"duration_seconds":    fmt.Sprintf("%.1f", svcResp.DurationSeconds),
		"findings":            svcResp.Findings,
	})

	// Advance pipeline — may dispatch next eligible services.
	s.AdvancePipeline(ctx, study.ID)
}
