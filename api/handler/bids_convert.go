package handler

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aegis-imaging/aegis/api/model"
)

// bidsConvertRequest is the payload sent to the Python BIDS service.
type bidsConvertRequest struct {
	StudyUID  string `json:"study_uid"`
	InputDir  string `json:"input_dir"`
	OutputDir string `json:"output_dir"`
}

// bidsConvertServiceResponse is the response from the Python BIDS service.
type bidsConvertServiceResponse struct {
	StudyUID        string   `json:"study_uid"`
	Status          string   `json:"status"` // "complete" | "failed"
	OutputFiles     []string `json:"output_files"`
	Warnings        []string `json:"warnings"`
	ToolUsed        string   `json:"tool_used"`
	DurationSeconds float64  `json:"duration_seconds"`
	Error           *string  `json:"error"`
}

// TriggerBidsConversion triggers the BIDS conversion pipeline for a study.
// It dispatches to the Python service asynchronously and returns 202 Accepted.
func (s *Server) TriggerBidsConversion(w http.ResponseWriter, r *http.Request) {
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

	if !study.BidsRequired {
		s.writeError(w, http.StatusBadRequest, "study does not require BIDS conversion")
		return
	}

	if study.BidsStatus == "converting" {
		s.writeError(w, http.StatusConflict, "BIDS conversion already in progress")
		return
	}

	if err := model.UpdateBidsStatus(r.Context(), s.db, study.ID, "converting"); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to update BIDS status")
		return
	}

	model.CreateAuditEntry(r.Context(), s.db, "bids_conversion.triggered", actorEmail(r), "study", study.ID, clientIP(r), map[string]any{
		"study_uid": studyUID,
	})

	if s.cfg.BidsServiceURL == "" {
		log.Printf("bids_conversion: BIDS_SERVICE_URL not set — study %s queued but not processed", studyUID)
		s.writeJSON(w, http.StatusAccepted, map[string]string{
			"status":  "converting",
			"message": "Study queued for BIDS conversion — set BIDS_SERVICE_URL to enable processing",
		})
		return
	}

	go s.runBidsConversion(study)

	s.writeJSON(w, http.StatusAccepted, map[string]string{
		"status":  "converting",
		"message": "BIDS conversion pipeline started",
	})
}

// runBidsConversion calls the Python BIDS service and updates the study record.
func (s *Server) runBidsConversion(study *model.Study) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	studyUID := study.StudyInstanceUID

	// Input: study's current DICOM store
	inputDir := filepath.Join(s.cfg.LocalStorageDir, "dicom", study.DicomStore, studyUID)
	if _, err := os.Stat(inputDir); err != nil {
		log.Printf("bids_conversion: input dir not found for %s: %v", studyUID, err)
		model.UpdateBidsStatus(ctx, s.db, study.ID, "failed")
		return
	}

	// Output: bids/{studyUID}/
	outputDir := filepath.Join(s.cfg.LocalStorageDir, "bids", studyUID)

	payload := bidsConvertRequest{
		StudyUID:  studyUID,
		InputDir:  inputDir,
		OutputDir: outputDir,
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		s.cfg.BidsServiceURL+"/convert", bytes.NewReader(body))
	if err != nil {
		log.Printf("bids_conversion: build request for %s: %v", studyUID, err)
		model.UpdateBidsStatus(ctx, s.db, study.ID, "failed")
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		log.Printf("bids_conversion: call service for %s: %v", studyUID, err)
		model.UpdateBidsStatus(ctx, s.db, study.ID, "failed")
		return
	}
	defer resp.Body.Close()

	var svcResp bidsConvertServiceResponse
	if err := json.NewDecoder(resp.Body).Decode(&svcResp); err != nil {
		log.Printf("bids_conversion: decode response for %s: %v", studyUID, err)
		model.UpdateBidsStatus(ctx, s.db, study.ID, "failed")
		return
	}

	if svcResp.Status != "complete" {
		errMsg := "unknown error"
		if svcResp.Error != nil {
			errMsg = *svcResp.Error
		}
		log.Printf("bids_conversion: service failed for %s (%s): %s", studyUID, svcResp.ToolUsed, errMsg)
		model.UpdateBidsStatus(ctx, s.db, study.ID, "failed")
		model.CreateAuditEntry(ctx, s.db, "bids_conversion.failed", "system", "study", study.ID, "", map[string]any{
			"study_uid": studyUID,
			"tool":      svcResp.ToolUsed,
			"error":     errMsg,
		})
		return
	}

	if err := model.UpdateBidsStatus(ctx, s.db, study.ID, "complete"); err != nil {
		log.Printf("bids_conversion: update study record for %s: %v", studyUID, err)
		return
	}

	log.Printf("bids_conversion: complete for %s — %d files, %d warnings in %.1fs using %s",
		studyUID, len(svcResp.OutputFiles), len(svcResp.Warnings), svcResp.DurationSeconds, svcResp.ToolUsed)

	model.CreateAuditEntry(ctx, s.db, "bids_conversion.complete", "system", "study", study.ID, "", map[string]any{
		"study_uid":        studyUID,
		"tool":             svcResp.ToolUsed,
		"output_files":     len(svcResp.OutputFiles),
		"warnings":         svcResp.Warnings,
		"duration_seconds": fmt.Sprintf("%.1f", svcResp.DurationSeconds),
	})

	// Advance pipeline — may dispatch next eligible services.
	s.AdvancePipeline(ctx, study.ID)
}

// ServeBidsDownload streams the BIDS output directory as a zip archive.
func (s *Server) ServeBidsDownload(w http.ResponseWriter, r *http.Request) {
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

	if study.BidsStatus != "complete" {
		s.writeError(w, http.StatusBadRequest, "BIDS conversion not complete")
		return
	}

	bidsDir := filepath.Join(s.cfg.LocalStorageDir, "bids", studyUID)
	if _, err := os.Stat(bidsDir); err != nil {
		s.writeError(w, http.StatusNotFound, "BIDS output not found on disk")
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s_bids.zip"`, studyUID))

	zw := zip.NewWriter(w)
	defer zw.Close()

	filepath.Walk(bidsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		relPath, _ := filepath.Rel(bidsDir, path)
		// Use forward slashes in zip entries
		relPath = strings.ReplaceAll(relPath, string(filepath.Separator), "/")

		fw, err := zw.Create(relPath)
		if err != nil {
			return err
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(fw, f)
		return err
	})
}
