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
//	minutes  — idle threshold in minutes (default: 60)
//	project_id — optional project UUID filter
func (s *Server) GetStuckStudies(w http.ResponseWriter, r *http.Request) {
	minutes := 60
	if m := r.URL.Query().Get("minutes"); m != "" {
		if n, err := strconv.Atoi(m); err == nil && n > 0 {
			minutes = n
		}
	}
	projectID := r.URL.Query().Get("project_id")

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
