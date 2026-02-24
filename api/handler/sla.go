package handler

import (
	"net/http"
	"strconv"

	"github.com/aegis-imaging/aegis/api/model"
)

// GetStuckStudies returns studies that are not in a terminal state and have not
// been updated within the configured SLA window.
//
// Query params:
//
//	minutes    — idle threshold in minutes (default: 60, or project's stuck_threshold_minutes if set)
//	project_id — optional project UUID filter
func (s *Server) GetStuckStudies(w http.ResponseWriter, r *http.Request) {
	minutes := 60
	projectID := r.URL.Query().Get("project_id")

	// If a project is scoped and it has a per-project threshold, use it as the default.
	if projectID != "" {
		if p, err := model.GetProjectByID(r.Context(), s.db, projectID); err == nil && p.StuckThresholdMinutes != nil {
			minutes = *p.StuckThresholdMinutes
		}
	}

	// Explicit ?minutes= param always wins (overrides project default).
	if m := r.URL.Query().Get("minutes"); m != "" {
		if n, err := strconv.Atoi(m); err == nil && n > 0 {
			minutes = n
		}
	}

	studies, err := model.GetStuckStudies(r.Context(), s.db, minutes, projectID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	if studies == nil {
		studies = []model.Study{}
	}
	s.writeJSON(w, http.StatusOK, map[string]any{
		"stuck":   studies,
		"total":   len(studies),
		"minutes": minutes,
	})
}
