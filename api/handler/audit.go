package handler

import (
	"net/http"
	"strconv"

	"github.com/aegis-imaging/aegis/api/model"
)

func (s *Server) ListAudit(w http.ResponseWriter, r *http.Request) {
	action := r.URL.Query().Get("action")
	resourceType := r.URL.Query().Get("resource_type")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	if limit <= 0 {
		limit = 100
	}

	entries, err := model.ListAuditEntries(r.Context(), s.db, action, resourceType, limit)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to list audit entries")
		return
	}
	if entries == nil {
		entries = []model.AuditEntry{}
	}
	s.writeJSON(w, http.StatusOK, entries)
}
