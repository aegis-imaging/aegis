package handler

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/aegis-imaging/aegis/api/middleware"
	"github.com/aegis-imaging/aegis/api/model"
)

// projectSubjectsResponse wraps the enriched subject list returned for a
// project. Demographics are merged in when a row exists in
// subject_demographics for that (project, subject) pair.
type projectSubjectsResponse struct {
	ProjectID string                  `json:"project_id"`
	Subjects  []model.SubjectAggregate `json:"subjects"`
	Total     int                     `json:"total"`
}

// ListProjectSubjects GET /api/projects/{id}/subjects
//
// Returns one row per subject_id within the project, with study count,
// latest study date, modalities, and (if present) research demographics.
// Access is enforced via requireProjectReadAccess so site-scoped researchers
// only see subjects whose studies live in their institution.
func (s *Server) ListProjectSubjects(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")

	access, ok := s.requireProjectReadAccess(w, r, projectID)
	if !ok {
		return
	}

	institutionID := ""
	if access != nil && access.IsSiteScoped() {
		institutionID = *access.InstitutionID
	}

	subjects, err := model.ListProjectSubjects(r.Context(), s.db, projectID, institutionID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to list subjects")
		return
	}
	if subjects == nil {
		subjects = []model.SubjectAggregate{}
	}

	// Bulk-load demographics for this project and merge by subject_id. This
	// keeps the response a single round trip even when most subjects have
	// demographics filled in.
	demos, err := model.ListProjectDemographics(r.Context(), s.db, projectID)
	if err == nil && len(demos) > 0 {
		bySubject := make(map[string]*model.SubjectDemographics, len(demos))
		for i := range demos {
			bySubject[demos[i].SubjectID] = &demos[i]
		}
		for i := range subjects {
			if d, ok := bySubject[subjects[i].SubjectID]; ok {
				subjects[i].Demographics = d
			}
		}
	}

	s.writeJSON(w, http.StatusOK, projectSubjectsResponse{
		ProjectID: projectID,
		Subjects:  subjects,
		Total:     len(subjects),
	})
}

// projectSubjectResponse bundles the subject aggregate, its demographics, and
// the list of studies for the subject on a single page-load.
type projectSubjectResponse struct {
	model.SubjectAggregate
	Studies []model.Study `json:"studies"`
}

// GetProjectSubject GET /api/projects/{id}/subjects/{subjectID}
//
// Returns the subject aggregate plus the full study list for that subject in
// the project, ordered by study_date desc. Used to power the Subject detail
// page in the XNAT-style nav.
func (s *Server) GetProjectSubject(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	subjectID := r.PathValue("subjectID")

	access, ok := s.requireProjectReadAccess(w, r, projectID)
	if !ok {
		return
	}

	institutionID := ""
	if access != nil && access.IsSiteScoped() {
		institutionID = *access.InstitutionID
	}

	agg, err := model.GetProjectSubject(r.Context(), s.db, projectID, subjectID, institutionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.writeError(w, http.StatusNotFound, "subject not found")
			return
		}
		s.writeError(w, http.StatusInternalServerError, "failed to load subject")
		return
	}

	if d, err := model.GetSubjectDemographics(r.Context(), s.db, subjectID, projectID); err == nil {
		agg.Demographics = d
	}

	f := model.StudyFilters{
		ProjectID: projectID,
		SubjectID: subjectID,
		SortBy:    "study_date",
		SortDir:   "desc",
	}
	if institutionID != "" {
		f.InstitutionID = institutionID
	}
	if t := middleware.TenantFromContext(r.Context()); t != nil {
		f.TenantID = t.ID
	}
	studies, err := model.ListStudies(r.Context(), s.db, f, 500, 0)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to list studies for subject")
		return
	}
	if studies == nil {
		studies = []model.Study{}
	}

	s.writeJSON(w, http.StatusOK, projectSubjectResponse{
		SubjectAggregate: *agg,
		Studies:          studies,
	})
}
