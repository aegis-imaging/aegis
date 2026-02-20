package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/msenjem/aegis/api/model"
	"github.com/msenjem/aegis/api/routing"
)

type ingestRequest struct {
	ProjectSlug        string        `json:"project_slug"`
	InstitutionID      string        `json:"institution_id,omitempty"`
	InstitutionAETitle string        `json:"institution_ae_title,omitempty"`
	Metadata           studyMetadata `json:"study_metadata"`
}

// InternalIngest accepts studies originating inside the enterprise network.
// It tags the study as source="internal" and enters the same processing pipeline
// as externally-uploaded studies (de-id validation, defacing gate, QC, approval).
//
// NOTE: In Phase 1 the pipeline steps are stubs — the study record is created
// and routed, but server-side processing is not yet wired up.
func (s *Server) InternalIngest(w http.ResponseWriter, r *http.Request) {
	var req ingestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	slug := req.ProjectSlug
	if slug == "" {
		slug = "default"
	}
	project, err := model.GetProjectBySlug(r.Context(), s.db, slug)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "project not found: "+slug)
		return
	}

	var institutionID *string
	if strings.TrimSpace(req.InstitutionID) != "" || strings.TrimSpace(req.InstitutionAETitle) != "" {
		inst, err := s.resolveIngestInstitution(r.Context(), project.ID, req.InstitutionID, req.InstitutionAETitle)
		if err != nil {
			s.writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		institutionID = &inst.ID
	}

	studyUID := req.Metadata.StudyInstanceUID
	if studyUID == "" {
		studyUID = fmt.Sprintf("2.25.%d", time.Now().UnixNano())
	}

	bodyPart := strings.ToUpper(req.Metadata.BodyPart)
	defacingRequired := bodyPart == "HEAD" || bodyPart == "BRAIN"

	study := &model.Study{
		ProjectID:        project.ID,
		StudyInstanceUID: studyUID,
		Modality:         req.Metadata.Modality,
		BodyPart:         req.Metadata.BodyPart,
		StudyDescription: req.Metadata.StudyDescription,
		SeriesCount:      req.Metadata.SeriesCount,
		InstanceCount:    req.Metadata.InstanceCount,
		Status:           "received",
		DefacingRequired: defacingRequired,
		DicomStore:       "raw",
		Source:           "internal",
		InstitutionID:    institutionID,
	}
	if err := model.CreateStudy(r.Context(), s.db, study); err != nil {
		log.Printf("create study (internal ingest): %v", err)
		s.writeError(w, http.StatusInternalServerError, "failed to create study record")
		return
	}

	// Evaluate routing rules — may mutate study (e.g. auto_approve, require_defacing).
	routing.EvaluateRules(r.Context(), s.db, s.store, study)

	// Auto-dispatch processing pipeline.
	s.AdvancePipeline(r.Context(), study.ID)

	detail := map[string]any{
		"study_uid": studyUID,
		"modality":  study.Modality,
		"project":   slug,
	}
	if institutionID != nil {
		detail["institution_id"] = *institutionID
	}
	if strings.TrimSpace(req.InstitutionAETitle) != "" {
		detail["institution_ae_title"] = strings.TrimSpace(req.InstitutionAETitle)
	}
	model.CreateAuditEntry(r.Context(), s.db, "ingest.internal", actorEmail(r), "study", study.ID, clientIP(r), detail)

	s.writeJSON(w, http.StatusCreated, map[string]any{
		"status":  "received",
		"study":   study,
		"message": "Study received — processing pipeline pending Phase 1 implementation",
	})
}

func normalizeAETitle(aeTitle string) string {
	return strings.ToUpper(strings.TrimSpace(aeTitle))
}

func (s *Server) resolveIngestInstitution(ctx context.Context, projectID, institutionID, institutionAETitle string) (*model.Institution, error) {
	var (
		inst *model.Institution
		err  error
	)

	institutionID = strings.TrimSpace(institutionID)
	institutionAETitle = strings.TrimSpace(institutionAETitle)

	if institutionID != "" {
		inst, err = model.GetInstitutionByID(ctx, s.db, institutionID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, fmt.Errorf("institution not found")
			}
			return nil, fmt.Errorf("lookup institution: %w", err)
		}
	}

	if institutionAETitle != "" {
		if inst == nil {
			inst, err = model.GetInstitutionByAETitle(ctx, s.db, institutionAETitle)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return nil, fmt.Errorf("institution not found for ae_title %q", institutionAETitle)
				}
				return nil, fmt.Errorf("lookup institution by ae_title: %w", err)
			}
		} else if normalizeAETitle(inst.AETitle) != normalizeAETitle(institutionAETitle) {
			return nil, fmt.Errorf("institution_id and institution_ae_title do not match")
		}
	}

	if inst == nil {
		return nil, fmt.Errorf("institution_id or institution_ae_title required")
	}
	if !inst.Enabled {
		return nil, fmt.Errorf("institution is disabled")
	}
	if inst.Type != "sender" && inst.Type != "both" {
		return nil, fmt.Errorf("institution type must be sender or both")
	}

	allowed, err := model.InstitutionCanSendToProject(ctx, s.db, inst.ID, projectID)
	if err != nil {
		return nil, fmt.Errorf("validate institution project link: %w", err)
	}
	if !allowed {
		return nil, fmt.Errorf("institution is not linked to project as sender/admin")
	}
	return inst, nil
}
