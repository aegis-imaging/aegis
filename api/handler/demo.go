package handler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/aegis-imaging/aegis/api/importer"
	"github.com/aegis-imaging/aegis/api/model"
)

// DemoGenerate generates a synthetic brain MRI and runs it through the full pipeline.
// The study is generated with facial anatomy (with_face=true) so the defacing step
// produces a visible before/after comparison.
//
// Public, rate-limited — no authentication required.
//
// POST /api/demo/generate
func (s *Server) DemoGenerate(w http.ResponseWriter, r *http.Request) {
	if s.cfg.SynthServiceURL == "" {
		s.writeError(w, http.StatusServiceUnavailable,
			"demo unavailable — synthetic MRI service not configured")
		return
	}

	// Fixed demo parameters: with_face=true so defacing produces a visible result.
	// Seed is randomised so each user gets a distinct phantom variant.
	payload := synthSvcRequest{
		Slices:   20,
		Size:     256,
		Seed:     rand.Intn(100000),
		WithFace: true,
		UseGPU:   false,
	}
	body, _ := json.Marshal(payload)

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
		log.Printf("demo_generate: synth service unavailable: %v", err)
		s.writeError(w, http.StatusBadGateway, "synth service unavailable")
		return
	}
	defer resp.Body.Close()

	var svcResp synthSvcResponse
	if err := json.NewDecoder(resp.Body).Decode(&svcResp); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to decode synth response")
		return
	}
	if svcResp.Error != nil {
		log.Printf("demo_generate: synth error: %s", *svcResp.Error)
		s.writeError(w, http.StatusInternalServerError, "synthetic generation failed")
		return
	}

	// Import the generated DICOM files into the default project via the standard path.
	result, err := importer.Run(r.Context(), s.db, s.store, importer.Options{
		Dir:         svcResp.OutputDir,
		ProjectSlug: "default",
		Source:      "internal",
	})
	if err != nil {
		log.Printf("demo_generate: import failed (dir=%s): %v", svcResp.OutputDir, err)
		s.writeError(w, http.StatusInternalServerError, "failed to import demo study")
		return
	}

	// Force defacing on the demo study so the landing page can show before/after.
	for _, studyID := range result.StudyIDs {
		if err := model.SetDefacingRequired(r.Context(), s.db, studyID, true); err != nil {
			log.Printf("demo_generate: set defacing_required failed for %s: %v", studyID, err)
		}
	}

	model.CreateAuditEntry(r.Context(), s.db, "study.demo_generated", clientIP(r), "study",
		svcResp.StudyUID, clientIP(r), map[string]any{
			"study_uid":  svcResp.StudyUID,
			"file_count": svcResp.FileCount,
			"tool":       svcResp.ToolUsed,
		})

	// Auto-dispatch the processing pipeline (will trigger defacing in phase 1).
	for _, studyID := range result.StudyIDs {
		s.AdvancePipeline(r.Context(), studyID)
	}

	studyID := ""
	if len(result.StudyIDs) > 0 {
		studyID = result.StudyIDs[0]
	}

	log.Printf("demo_generate: created study %s from synthetic MRI (tool=%s, slices=20, %.2fs)",
		svcResp.StudyUID, svcResp.ToolUsed, svcResp.DurationSeconds)

	s.writeJSON(w, http.StatusOK, map[string]any{
		"study_id":  studyID,
		"study_uid": svcResp.StudyUID,
	})
}

// DemoStudyStatus returns the minimal status of a demo study for frontend polling.
// Public, rate-limited — no authentication required.
//
// GET /api/demo/study/{studyID}
func (s *Server) DemoStudyStatus(w http.ResponseWriter, r *http.Request) {
	studyID := r.PathValue("studyID")
	if studyID == "" {
		s.writeError(w, http.StatusBadRequest, "missing study ID")
		return
	}

	var (
		studyUID         string
		status           string
		defacingRequired bool
		dicomStore       string
	)
	err := s.db.QueryRowContext(r.Context(),
		`SELECT study_instance_uid, status, defacing_required, dicom_store
		   FROM studies WHERE id = $1`,
		studyID,
	).Scan(&studyUID, &status, &defacingRequired, &dicomStore)

	if err == sql.ErrNoRows {
		s.writeError(w, http.StatusNotFound, "study not found")
		return
	}
	if err != nil {
		log.Printf("demo_status: db error for %s: %v", studyID, err)
		s.writeError(w, http.StatusInternalServerError, "database error")
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"study_id":          studyID,
		"study_uid":         studyUID,
		"status":            status,
		"defacing_required": defacingRequired,
		"dicom_store":       dicomStore,
	})
}
