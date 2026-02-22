package handler

import (
	"log"
	"net/http"

	"github.com/aegis-imaging/aegis/api/model"
)

// ExportBatch dispatches export forwarding for all eligible approved studies
// in a project (status=approved, export_required=true, export_status in pending/failed).
//
// POST /api/projects/{id}/export-batch
func (s *Server) ExportBatch(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")

	// Load all approved studies in the project that need exporting.
	rows, err := s.db.QueryContext(r.Context(), `
		SELECT id, study_instance_uid
		FROM studies
		WHERE project_id = $1
		  AND status = 'approved'
		  AND export_required = true
		  AND export_status IN ('pending', 'failed')
		ORDER BY created_at ASC`, projectID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to query studies")
		return
	}
	defer rows.Close()

	type candidate struct {
		id  string
		uid string
	}
	var candidates []candidate
	for rows.Next() {
		var c candidate
		if err := rows.Scan(&c.id, &c.uid); err != nil {
			continue
		}
		candidates = append(candidates, c)
	}
	if err := rows.Err(); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to scan studies")
		return
	}

	if len(candidates) == 0 {
		s.writeJSON(w, http.StatusOK, map[string]any{
			"dispatched": 0,
			"study_ids":  []string{},
			"message":    "No eligible studies found for export in this project",
		})
		return
	}

	// Reset any 'failed' back to 'pending' and claim each study.
	var dispatched []string
	for _, c := range candidates {
		// Idempotent reset for failed studies.
		model.UpdateExportStatus(r.Context(), s.db, c.id, "pending")

		claimed, err := model.ClaimExport(r.Context(), s.db, c.id)
		if err != nil || !claimed {
			log.Printf("export_batch: could not claim %s: %v", c.uid, err)
			continue
		}
		study, err := model.GetStudyByID(r.Context(), s.db, c.id)
		if err != nil {
			model.UpdateExportStatus(r.Context(), s.db, c.id, "pending")
			continue
		}
		go s.runExportForward(study)
		dispatched = append(dispatched, c.id)
	}

	model.CreateAuditEntry(r.Context(), s.db, "export.batch_dispatched", actorEmail(r), "project", projectID, clientIP(r), map[string]any{
		"dispatched_count": len(dispatched),
		"candidate_count":  len(candidates),
	})

	if dispatched == nil {
		dispatched = []string{}
	}
	s.writeJSON(w, http.StatusAccepted, map[string]any{
		"dispatched": len(dispatched),
		"study_ids":  dispatched,
		"message":    "Export forwarding dispatched",
	})
}
