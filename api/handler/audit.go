package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/aegis-imaging/aegis/api/middleware"
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
	// Tenant scope: a tenant in the request context restricts the listing to
	// that tenant's audit rows. Legacy single-tenant requests (no tenant)
	// see everything they would have seen pre-multitenant.
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

	var total int
	var err error
	if access != nil {
		total, err = model.CountAuditEntriesForStudyScope(r.Context(), s.db, f, access.ProjectID, institutionID)
	} else {
		total, err = model.CountAuditEntries(r.Context(), s.db, f)
	}
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to count audit entries")
		return
	}

	var entries []model.AuditEntry
	if access != nil {
		entries, err = model.ListAuditEntriesForStudyScope(r.Context(), s.db, f, access.ProjectID, institutionID, limit, offset)
	} else {
		entries, err = model.ListAuditEntries(r.Context(), s.db, f, limit, offset)
	}
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

// GetAuditActors returns recent admin actors with their last-seen timestamp and action count.
// GET /api/audit/actors
func (s *Server) GetAuditActors(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	projectID := r.URL.Query().Get("project_id")
	access, ok := s.requireResearcherProjectScope(w, r, projectID)
	if !ok {
		return
	}
	institutionID := ""
	if access != nil && access.IsSiteScoped() {
		institutionID = *access.InstitutionID
	}

	var actors []model.ActorSummary
	var err error
	if access != nil {
		actors, err = model.GetActorSummaryForStudyScope(r.Context(), s.db, access.ProjectID, institutionID, limit)
	} else {
		actors, err = model.GetActorSummary(r.Context(), s.db, limit)
	}
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to query actor summary")
		return
	}
	if actors == nil {
		actors = []model.ActorSummary{}
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"actors": actors})
}
