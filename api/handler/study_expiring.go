package handler

import (
	"net/http"
	"strconv"

	"github.com/aegis-imaging/aegis/api/model"
)

// GetExpiringStudies returns approved studies that will be soft-expired within
// the given number of days based on their project's retention policy.
//
// GET /api/studies/expiring?days=7&project_id=<uuid>
//
// Only projects with a non-null retention_days are considered. Studies where
// created_at + retention_days falls within (now, now+days] are returned,
// ordered soonest-first.
func (s *Server) GetExpiringStudies(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	projectID := q.Get("project_id")

	days := 7
	if d := q.Get("days"); d != "" {
		if n, err := strconv.Atoi(d); err == nil && n > 0 && n <= 365 {
			days = n
		}
	}

	limit := 200
	if l := q.Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}

	studies, err := model.GetExpiringStudies(r.Context(), s.db, projectID, days, limit)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "expiring studies query failed")
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"studies":   studies,
		"total":     len(studies),
		"days":      days,
		"truncated": len(studies) == limit,
	})
}
