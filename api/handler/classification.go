package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/msenjem/aegis/api/model"
	"github.com/msenjem/aegis/api/routing"
)

// classifyRequest is the payload sent to the Python classification service.
type classifyRequest struct {
	StudyUID   string   `json:"study_uid"`
	InputPaths []string `json:"input_paths"`
}

// classifyServiceResponse is the response from the Python classification service.
type classifyServiceResponse struct {
	Status          string  `json:"status"` // "complete" | "failed"
	Modality        string  `json:"modality"`
	BodyPart        string  `json:"body_part"`
	Confidence      float64 `json:"confidence"`
	Method          string  `json:"method"`
	ToolUsed        string  `json:"tool_used"`
	DurationSeconds float64 `json:"duration_seconds"`
	Error           *string `json:"error"`
}

// TriggerClassification triggers the metadata classification pipeline for a study.
// It dispatches to the Python service asynchronously and returns 202 Accepted.
func (s *Server) TriggerClassification(w http.ResponseWriter, r *http.Request) {
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

	if !study.ClassificationRequired {
		s.writeError(w, http.StatusBadRequest, "study does not require classification")
		return
	}

	if study.ClassificationStatus == "classifying" {
		s.writeError(w, http.StatusConflict, "classification already in progress")
		return
	}

	if err := model.UpdateClassificationStatus(r.Context(), s.db, study.ID, "classifying"); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to update classification status")
		return
	}

	model.CreateAuditEntry(r.Context(), s.db, "classification.triggered", actorEmail(r), "study", study.ID, clientIP(r), map[string]any{
		"study_uid": studyUID,
	})

	if s.cfg.ClassificationServiceURL == "" {
		log.Printf("classification: CLASSIFICATION_SERVICE_URL not set — study %s queued but not processed", studyUID)
		s.writeJSON(w, http.StatusAccepted, map[string]string{
			"status":  "classifying",
			"message": "Study queued for classification — set CLASSIFICATION_SERVICE_URL to enable processing",
		})
		return
	}

	go s.runClassification(study)

	s.writeJSON(w, http.StatusAccepted, map[string]string{
		"status":  "classifying",
		"message": "Classification pipeline started",
	})
}

// runClassification calls the Python classification service, updates study metadata,
// and re-evaluates routing rules so modality/body_part-dependent rules can fire.
func (s *Server) runClassification(study *model.Study) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	studyUID := study.StudyInstanceUID

	files, err := s.listDicomFiles("raw", studyUID)
	if err != nil || len(files) == 0 {
		log.Printf("classification: no files found for study %s: %v", studyUID, err)
		model.UpdateClassificationStatus(ctx, s.db, study.ID, "failed")
		return
	}

	payload := classifyRequest{
		StudyUID:   studyUID,
		InputPaths: files,
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		s.cfg.ClassificationServiceURL+"/classify", bytes.NewReader(body))
	if err != nil {
		log.Printf("classification: build request for %s: %v", studyUID, err)
		model.UpdateClassificationStatus(ctx, s.db, study.ID, "failed")
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("classification: call service for %s: %v", studyUID, err)
		model.UpdateClassificationStatus(ctx, s.db, study.ID, "failed")
		return
	}
	defer resp.Body.Close()

	var svcResp classifyServiceResponse
	if err := json.NewDecoder(resp.Body).Decode(&svcResp); err != nil {
		log.Printf("classification: decode response for %s: %v", studyUID, err)
		model.UpdateClassificationStatus(ctx, s.db, study.ID, "failed")
		return
	}

	if svcResp.Status != "complete" {
		errMsg := "unknown error"
		if svcResp.Error != nil {
			errMsg = *svcResp.Error
		}
		log.Printf("classification: service failed for %s (%s): %s", studyUID, svcResp.ToolUsed, errMsg)
		model.UpdateClassificationStatus(ctx, s.db, study.ID, "failed")
		model.CreateAuditEntry(ctx, s.db, "classification.failed", "system", "study", study.ID, "", map[string]any{
			"study_uid": studyUID,
			"tool":      svcResp.ToolUsed,
			"error":     errMsg,
		})
		return
	}

	// Update study metadata if the classification found modality/body_part with sufficient confidence.
	metadataUpdated := false
	newModality := svcResp.Modality
	newBodyPart := svcResp.BodyPart
	if svcResp.Confidence >= 0.5 && (newModality != "" || newBodyPart != "") {
		// Only update fields that were empty or if classified with high confidence.
		if study.Modality == "" && newModality != "" {
			study.Modality = newModality
		}
		if study.BodyPart == "" && newBodyPart != "" {
			study.BodyPart = newBodyPart
		}
		if err := model.UpdateStudyMetadata(ctx, s.db, study.ID, study.Modality, study.BodyPart); err != nil {
			log.Printf("classification: update metadata for %s: %v", studyUID, err)
		} else {
			metadataUpdated = true
		}
	}

	if err := model.UpdateClassificationStatus(ctx, s.db, study.ID, "classified"); err != nil {
		log.Printf("classification: update status for %s: %v", studyUID, err)
		return
	}

	log.Printf("classification: complete for %s — modality=%s body_part=%s confidence=%.2f method=%s in %.1fs using %s",
		studyUID, study.Modality, study.BodyPart, svcResp.Confidence, svcResp.Method,
		svcResp.DurationSeconds, svcResp.ToolUsed)

	model.CreateAuditEntry(ctx, s.db, "classification.complete", "system", "study", study.ID, "", map[string]any{
		"study_uid":        studyUID,
		"tool":             svcResp.ToolUsed,
		"modality":         study.Modality,
		"body_part":        study.BodyPart,
		"confidence":       svcResp.Confidence,
		"method":           svcResp.Method,
		"metadata_updated": metadataUpdated,
		"duration_seconds": svcResp.DurationSeconds,
	})

	// Re-evaluate routing rules now that modality/body_part are filled in.
	// This allows downstream rules (e.g. require_defacing for HEAD studies) to fire.
	if metadataUpdated {
		// Re-fetch the study to get the latest state after classification updates.
		updated, err := model.GetStudyByID(ctx, s.db, study.ID)
		if err != nil {
			log.Printf("classification: re-fetch study %s for routing re-evaluation: %v", studyUID, err)
			return
		}
		log.Printf("classification: re-evaluating routing rules for %s (modality=%s, body_part=%s)",
			studyUID, updated.Modality, updated.BodyPart)
		routing.EvaluateRules(ctx, s.db, updated)
	}
}
