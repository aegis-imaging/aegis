package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/msenjem/aegis/api/model"
)

type ingestRequest struct {
	ProjectSlug string        `json:"project_slug"`
	Metadata    studyMetadata `json:"study_metadata"`
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
	}
	if err := model.CreateStudy(r.Context(), s.db, study); err != nil {
		log.Printf("create study (internal ingest): %v", err)
		s.writeError(w, http.StatusInternalServerError, "failed to create study record")
		return
	}

	model.CreateAuditEntry(r.Context(), s.db, "ingest.internal", "internal", "study", study.ID, clientIP(r), map[string]any{
		"study_uid": studyUID,
		"modality":  study.Modality,
		"project":   slug,
	})

	s.writeJSON(w, http.StatusCreated, map[string]any{
		"status":  "received",
		"study":   study,
		"message": "Study received — processing pipeline pending Phase 1 implementation",
	})
}
