package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/aegis-imaging/aegis/api/middleware"
	"github.com/aegis-imaging/aegis/api/model"
)

type studiesResponse struct {
	Studies []model.Study `json:"studies"`
	Total   int           `json:"total"`
	Limit   int           `json:"limit"`
	Offset  int           `json:"offset"`
}

func (s *Server) ListStudies(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
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

	f := model.StudyFilters{
		ProjectID:     q.Get("project_id"),
		Status:        q.Get("status"),
		Modality:      q.Get("modality"),
		BodyPart:      q.Get("body_part"),
		Source:        q.Get("source"),
		Search:        q.Get("search"),
		SubjectID:     q.Get("subject_id"),
		Label:         q.Get("label"),
		InstitutionID: q.Get("institution_id"),
		DateFrom:      dateFrom,
		DateTo:        dateTo,
		StudyDateFrom: q.Get("study_date_from"),
		StudyDateTo:   q.Get("study_date_to"),
		SortBy:        q.Get("sort_by"),
		SortDir:       q.Get("sort_dir"),
	}
	if v := q.Get("flagged"); v == "true" {
		t := true
		f.Flagged = &t
	}
	if v := q.Get("assigned_to"); v != "" {
		f.AssignedTo = v
	}

	access, ok := s.requireResearcherProjectScope(w, r, f.ProjectID)
	if !ok {
		return
	}
	if access != nil && access.IsSiteScoped() {
		f.InstitutionID = *access.InstitutionID
	}
	// Tenant scoping: when the request resolved a tenant, restrict the
	// query to studies whose parent project belongs to that tenant.
	// Legacy requests (no tenant context) skip this clause and see
	// everything they would have seen pre-multitenant.
	if t := middleware.TenantFromContext(r.Context()); t != nil {
		f.TenantID = t.ID
	}

	total, err := model.CountStudies(r.Context(), s.db, f)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to count studies")
		return
	}

	studies, err := model.ListStudies(r.Context(), s.db, f, limit, offset)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to list studies")
		return
	}
	if studies == nil {
		studies = []model.Study{}
	}

	s.writeJSON(w, http.StatusOK, studiesResponse{
		Studies: studies,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
	})
}

// GetStudy returns a single study by ID.
func (s *Server) GetStudy(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	study, _, ok := s.requireStudyReadAccessByID(w, r, id)
	if !ok {
		return
	}
	s.writeJSON(w, http.StatusOK, study)
}

// GetStudyByUID returns a single study by DICOM StudyInstanceUID.
// Useful for integrations that only have the DICOM UID and not the DB UUID.
func (s *Server) GetStudyByUID(w http.ResponseWriter, r *http.Request) {
	uid := r.PathValue("studyUID")
	study, _, ok := s.requireStudyReadAccessByUID(w, r, uid)
	if !ok {
		return
	}
	s.writeJSON(w, http.StatusOK, study)
}

// DeleteStudy permanently deletes a study, its DICOM files, and all child rows.
// This is a destructive, irreversible action — admin-only.
func (s *Server) DeleteStudy(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	study, err := model.GetStudyByID(r.Context(), s.db, id)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "study not found")
		return
	}

	// Delete DICOM files from storage (best-effort — don't block DB delete on storage errors).
	for _, prefix := range []string{
		"dicom/raw/" + study.StudyInstanceUID + "/",
		"dicom/clean/" + study.StudyInstanceUID + "/",
		"bids/" + study.StudyInstanceUID + "/",
	} {
		if keys, lerr := s.store.List(r.Context(), prefix); lerr == nil {
			for _, k := range keys {
				_ = s.store.Delete(r.Context(), k)
			}
		}
	}

	if err := model.DeleteStudy(r.Context(), s.db, id); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to delete study")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "study.deleted", actorEmail(r), "study", id, clientIP(r), nil)
	w.WriteHeader(http.StatusNoContent)
}

// RecordStudyView records that an authenticated user opened a study detail panel.
// POST /api/studies/{id}/viewed
// Fire-and-forget from the dashboard; creates a study.viewed audit entry.
func (s *Server) RecordStudyView(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, _, ok := s.requireStudyReadAccessByID(w, r, id); !ok {
		return
	}
	actor := actorEmail(r)
	ip := clientIP(r)
	model.CreateAuditEntry(r.Context(), s.db, "study.viewed", actor, "study", id, ip, nil)
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ListStudyAudit returns all audit entries for a specific study.
func (s *Server) ListStudyAudit(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, _, ok := s.requireStudyReadAccessByID(w, r, id); !ok {
		return
	}

	entries, err := model.ListAuditEntriesForStudy(r.Context(), s.db, id)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to list audit entries")
		return
	}
	if entries == nil {
		entries = []model.AuditEntry{}
	}
	s.writeJSON(w, http.StatusOK, entries)
}
