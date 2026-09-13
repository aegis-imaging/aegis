package handler

import (
	"encoding/csv"
	"encoding/json"
	"net/http"
	"time"

	"github.com/aegis-imaging/aegis/api/middleware"
	"github.com/aegis-imaging/aegis/api/model"
)

// ExportAuditCSV streams audit trail entries matching the current filters as a CSV file.
// Accepts the same filter query params as GET /api/audit.
// Capped at 10 000 rows.
func (s *Server) ExportAuditCSV(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	var dateFrom, dateTo time.Time
	if v := q.Get("date_from"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			dateFrom = t.UTC()
		}
	}
	if v := q.Get("date_to"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			dateTo = t.UTC()
		}
	}

	f := model.AuditFilters{
		Action:       q.Get("action"),
		ResourceType: q.Get("resource_type"),
		Actor:        q.Get("actor"),
		Search:       q.Get("search"),
		DateFrom:     dateFrom,
		DateTo:       dateTo,
	}
	if t := middleware.TenantFromContext(r.Context()); t != nil {
		f.TenantID = t.ID
	}
	projectID := q.Get("project_id")
	access, ok := s.requireResearcherProjectScope(w, r, projectID)
	if !ok {
		return
	}
	institutionID := ""
	if access != nil && access.IsSiteScoped() {
		institutionID = *access.InstitutionID
	}

	var entries []model.AuditEntry
	var err error
	if access != nil {
		entries, err = model.ListAuditEntriesForStudyScope(r.Context(), s.db, f, access.ProjectID, institutionID, 10000, 0)
	} else {
		entries, err = model.ListAuditEntries(r.Context(), s.db, f, 10000, 0)
	}
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to list audit entries")
		return
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", `attachment; filename="audit.csv"`)
	w.WriteHeader(http.StatusOK)

	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"id", "created_at", "action", "actor", "resource_type", "resource_id", "ip_address", "detail"})

	for _, e := range entries {
		detail := ""
		if len(e.Detail) > 0 && string(e.Detail) != "null" {
			b, _ := json.Marshal(json.RawMessage(e.Detail))
			detail = string(b)
		}
		_ = cw.Write([]string{
			e.ID,
			e.CreatedAt.UTC().Format(time.RFC3339),
			e.Action,
			e.Actor,
			e.ResourceType,
			e.ResourceID,
			e.IPAddress,
			detail,
		})
	}

	cw.Flush()
}
