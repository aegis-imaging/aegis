package handler

import (
	"net/http"
	"strconv"

	"github.com/aegis-imaging/aegis/api/model"
)

// GetRoutingRuleChangelog GET /api/routing-rules/{id}/changelog
// Returns the modification history for a routing rule.
func (s *Server) GetRoutingRuleChangelog(w http.ResponseWriter, r *http.Request) {
	ruleID := r.PathValue("id")

	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}

	entries, err := model.ListRoutingRuleChangelog(r.Context(), s.db, ruleID, limit)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	if entries == nil {
		entries = []model.RoutingRuleChangelogEntry{}
	}
	s.writeJSON(w, http.StatusOK, map[string]any{
		"entries": entries,
		"total":   len(entries),
	})
}
