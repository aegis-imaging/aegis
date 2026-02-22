package handler

import (
	"net/http"
	"strconv"

	"github.com/aegis-imaging/aegis/api/model"
)

type auditResponse struct {
	Entries []model.AuditEntry `json:"entries"`
	Total   int                `json:"total"`
	Limit   int                `json:"limit"`
	Offset  int                `json:"offset"`
}

func (s *Server) ListAudit(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}

	f := model.AuditFilters{
		Action:       q.Get("action"),
		ResourceType: q.Get("resource_type"),
		Actor:        q.Get("actor"),
	}

	total, err := model.CountAuditEntries(r.Context(), s.db, f)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to count audit entries")
		return
	}

	entries, err := model.ListAuditEntries(r.Context(), s.db, f, limit, offset)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to list audit entries")
		return
	}
	if entries == nil {
		entries = []model.AuditEntry{}
	}
	s.writeJSON(w, http.StatusOK, auditResponse{
		Entries: entries,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
	})
}
