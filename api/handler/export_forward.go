package handler

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/msenjem/aegis/api/model"
	"github.com/msenjem/aegis/api/routing"
)

// TriggerExport manually triggers export forwarding for an approved study.
// The study must have export_required=true and export_status=pending.
func (s *Server) TriggerExport(w http.ResponseWriter, r *http.Request) {
	studyUID := r.PathValue("studyUID")
	study, err := model.GetStudyByUID(r.Context(), s.db, studyUID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "study not found")
		return
	}
	if study.Status != "approved" {
		s.writeError(w, http.StatusBadRequest, "only approved studies can be exported")
		return
	}
	if !study.ExportRequired {
		s.writeError(w, http.StatusBadRequest, "export not required for this study")
		return
	}
	if study.ExportStatus != "pending" && study.ExportStatus != "failed" {
		s.writeError(w, http.StatusBadRequest, "export already "+study.ExportStatus)
		return
	}

	// Reset to pending if retrying after failure, then claim.
	if study.ExportStatus == "failed" {
		if err := model.UpdateExportStatus(r.Context(), s.db, study.ID, "pending"); err != nil {
			s.writeError(w, http.StatusInternalServerError, "failed to reset export status")
			return
		}
	}

	claimed, err := model.ClaimExport(r.Context(), s.db, study.ID)
	if err != nil {
		log.Printf("export: claim export for %s: %v", study.StudyInstanceUID, err)
		s.writeError(w, http.StatusInternalServerError, "failed to claim export")
		return
	}
	if !claimed {
		s.writeError(w, http.StatusConflict, "export already in progress")
		return
	}

	model.CreateAuditEntry(r.Context(), s.db, "export.triggered", actorEmail(r),
		"study", study.ID, clientIP(r), map[string]any{
			"study_uid": studyUID,
		})

	go s.runExportForward(study)

	s.writeJSON(w, http.StatusAccepted, map[string]string{
		"status":    "exporting",
		"study_uid": studyUID,
	})
}

// runExportForward finds all matching route_to destinations and forwards the
// study's DICOM files via STOW-RS. Updates export_status on completion.
func (s *Server) runExportForward(study *model.Study) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	rules, err := model.ListEnabledRoutingRules(ctx, s.db)
	if err != nil {
		log.Printf("export: load rules for %s: %v", study.StudyInstanceUID, err)
		model.UpdateExportStatus(ctx, s.db, study.ID, "failed")
		return
	}

	var destinations []*model.Destination
	for i := range rules {
		r := &rules[i]
		if r.Action != "route_to" || r.DestinationID == nil {
			continue
		}
		if !routing.Matches(r, study) {
			continue
		}
		dest, err := model.GetDestinationByID(ctx, s.db, *r.DestinationID)
		if err != nil || !dest.Enabled {
			continue
		}
		destinations = append(destinations, dest)
	}

	if len(destinations) == 0 {
		log.Printf("export: no matching destinations for %s", study.StudyInstanceUID)
		model.UpdateExportStatus(ctx, s.db, study.ID, "failed")
		model.CreateAuditEntry(ctx, s.db, "export.failed", "export-engine",
			"study", study.ID, "", map[string]any{
				"study_uid": study.StudyInstanceUID,
				"error":     "no matching route_to destinations",
			})
		return
	}

	allOK := true
	for _, dest := range destinations {
		err := routing.ForwardStudy(ctx, s.db, s.store, study, dest)
		if err != nil {
			log.Printf("export: forward to %s failed for %s: %v", dest.Name, study.StudyInstanceUID, err)
			allOK = false
		}
	}

	if allOK {
		model.UpdateExportStatus(ctx, s.db, study.ID, "exported")
		model.CreateAuditEntry(ctx, s.db, "export.complete", "export-engine",
			"study", study.ID, "", map[string]any{
				"study_uid":         study.StudyInstanceUID,
				"destination_count": len(destinations),
			})
	} else {
		model.UpdateExportStatus(ctx, s.db, study.ID, "failed")
		model.CreateAuditEntry(ctx, s.db, "export.failed", "export-engine",
			"study", study.ID, "", map[string]any{
				"study_uid":         study.StudyInstanceUID,
				"destination_count": len(destinations),
				"error":             "one or more destinations failed",
			})
	}

	log.Printf("export: forward complete for %s (destinations=%d, success=%v)",
		study.StudyInstanceUID, len(destinations), allOK)
}
