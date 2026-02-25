package handler

import (
	"log"
	"net/http"
	"strconv"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/routing"
)

// BulkReEvaluateRouting re-runs all enabled routing rules against every study
// in the project that matches the optional status filter.
//
// POST /api/projects/{id}/re-evaluate-routing?status=received&limit=500
//
// Useful after updating routing rules to retroactively apply the new rules
// to existing studies without re-uploading them.
// Each study that gains a new pipeline requirement will be advanced by the
// auto-pipeline goroutine if PIPELINE_AUTO=true.
func (s *Server) BulkReEvaluateRouting(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	if projectID == "" {
		s.writeError(w, http.StatusBadRequest, "project_id required")
		return
	}

	ctx := r.Context()

	// Verify project exists.
	var exists bool
	if err := s.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM projects WHERE id = $1)`, projectID,
	).Scan(&exists); err != nil || !exists {
		s.writeError(w, http.StatusNotFound, "project not found")
		return
	}

	// Optional status filter (comma-separated values accepted but we take a single value for simplicity).
	statusFilter := r.URL.Query().Get("status")

	// Limit: default 500, max 2000.
	limit := 500
	if lv := r.URL.Query().Get("limit"); lv != "" {
		if n, err := strconv.Atoi(lv); err == nil && n > 0 {
			limit = n
		}
	}
	if limit > 2000 {
		limit = 2000
	}

	filters := model.StudyFilters{
		ProjectID: projectID,
		Status:    statusFilter,
	}
	studies, err := model.ListStudies(ctx, s.db, filters, limit, 0)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to list studies: "+err.Error())
		return
	}

	var evalErrors []string
	for i := range studies {
		func() {
			defer func() {
				if rec := recover(); rec != nil {
					evalErrors = append(evalErrors, studies[i].ID+": panic during routing")
					log.Printf("bulk-reeval: panic for study %s: %v", studies[i].ID, rec)
				}
			}()
			routing.EvaluateRules(ctx, s.db, s.store, &studies[i])
		}()
	}

	model.CreateAuditEntry(ctx, s.db, "routing.bulk_reeval", actorEmail(r), "project", projectID, clientIP(r),
		map[string]any{"evaluated": len(studies), "status_filter": statusFilter, "limit": limit, "errors": len(evalErrors)})

	s.writeJSON(w, http.StatusOK, map[string]any{
		"evaluated":     len(studies),
		"status_filter": statusFilter,
		"errors":        evalErrors,
	})
}
