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

// synthSvcRequest is the payload sent to the Python synth-service.
type synthSvcRequest struct {
	Slices   int    `json:"slices"`
	Size     int    `json:"size"`
	Seed     int    `json:"seed"`
	WithFace bool   `json:"with_face"`
	UseGPU   bool   `json:"use_gpu"`
}

// synthSvcResponse is the response from the Python synth-service.
type synthSvcResponse struct {
	StudyUID        string  `json:"study_uid"`
	OutputDir       string  `json:"output_dir"`
	FileCount       int     `json:"file_count"`
	DurationSeconds float64 `json:"duration_seconds"`
	ToolUsed        string  `json:"tool_used"`
	Error           *string `json:"error"`
}

// synthGenerateRequest is the incoming HTTP request body for this endpoint.
type synthGenerateRequest struct {
	ProjectSlug string `json:"project_slug"`
	Slices      int    `json:"slices"`
	Size        int    `json:"size"`
	Seed        int    `json:"seed"`
	WithFace    bool   `json:"with_face"`
	UseGPU      bool   `json:"use_gpu"`
}

// GenerateSyntheticStudy calls the synth sidecar to generate a synthetic brain MRI
// DICOM study and imports it into the project via the normal batch import path.
//
// POST /api/studies/generate-synthetic
func (s *Server) GenerateSyntheticStudy(w http.ResponseWriter, r *http.Request) {
	var req synthGenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if req.ProjectSlug == "" {
		req.ProjectSlug = "default"
	}
	if req.Slices <= 0 {
		req.Slices = 20
	}
	if req.Size <= 0 {
		req.Size = 256
	}

	if s.cfg.SynthServiceURL == "" {
		s.writeError(w, http.StatusServiceUnavailable,
			"synthetic MRI service not configured — set SYNTH_SERVICE_URL to enable")
		return
	}

	// Build the request to the Python sidecar.
	svcPayload := synthSvcRequest{
		Slices:   req.Slices,
		Size:     req.Size,
		Seed:     req.Seed,
		WithFace: req.WithFace,
		UseGPU:   req.UseGPU,
	}
	body, _ := json.Marshal(svcPayload)

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		s.cfg.SynthServiceURL+"/generate", bytes.NewReader(body))
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to build synth request")
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		log.Printf("synth_generate: call synth-service: %v", err)
		s.writeError(w, http.StatusBadGateway, "synth service unavailable")
		return
	}
	defer resp.Body.Close()

	var svcResp synthSvcResponse
	if err := json.NewDecoder(resp.Body).Decode(&svcResp); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to decode synth service response")
		return
	}

	if svcResp.Error != nil {
		log.Printf("synth_generate: service returned error: %s", *svcResp.Error)
		s.writeError(w, http.StatusInternalServerError,
			fmt.Sprintf("synthetic generation failed: %s", *svcResp.Error))
		return
	}

	// Import the generated DICOM files via the storage client (not GCS FUSE)
	// to avoid stale stat-cache issues with freshly-written objects.
	result, err := importSynthStudy(r.Context(), s.db, s.store, svcResp.StudyUID, req.ProjectSlug)
	if err != nil {
		log.Printf("synth_generate: import failed (uid=%s): %v", svcResp.StudyUID, err)
		s.writeError(w, http.StatusInternalServerError, "failed to import generated study")
		return
	}

	model.CreateAuditEntry(r.Context(), s.db, "study.synthetic_generated", actorEmail(r), "study",
		svcResp.StudyUID, clientIP(r), map[string]any{
			"study_uid":       svcResp.StudyUID,
			"tool":            svcResp.ToolUsed,
			"slices":          req.Slices,
			"size":            req.Size,
			"with_face":       req.WithFace,
			"use_gpu":         req.UseGPU,
			"file_count":      svcResp.FileCount,
			"study_ids":       result.StudyIDs,
			"duration_s":      fmt.Sprintf("%.2f", svcResp.DurationSeconds),
		})

	// Auto-dispatch processing pipeline for the new study.
	for _, studyID := range result.StudyIDs {
		s.AdvancePipeline(r.Context(), studyID)
	}

	log.Printf("synth_generate: created %d stud(ies) from synthetic MRI (tool=%s, slices=%d, %.2fs)",
		result.StudiesCreated, svcResp.ToolUsed, req.Slices, svcResp.DurationSeconds)

	s.writeJSON(w, http.StatusOK, map[string]any{
		"study_uid":        svcResp.StudyUID,
		"study_ids":        result.StudyIDs,
		"file_count":       svcResp.FileCount,
		"tool_used":        svcResp.ToolUsed,
		"duration_seconds": svcResp.DurationSeconds,
		"message":          fmt.Sprintf("Synthetic study created with %d DICOM files", svcResp.FileCount),
	})
}
