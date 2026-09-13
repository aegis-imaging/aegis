package handler

import (
	"net/http"
	"strconv"

	"github.com/aegis-imaging/aegis/api/model"
)

// GetInstitutionActivity GET /api/institutions/{id}/activity
// Returns recent audit entries related to an institution.
func (s *Server) GetInstitutionActivity(w http.ResponseWriter, r *http.Request) {
	instID := r.PathValue("id")

	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}

	entries, err := model.ListAuditEntriesForStudy(r.Context(), s.db, instID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	if entries == nil {
		entries = []model.AuditEntry{}
	}
	if len(entries) > limit {
		entries = entries[:limit]
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"entries": entries,
		"total":   len(entries),
	})
}
