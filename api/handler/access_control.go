package handler

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"github.com/aegis-imaging/aegis/api/middleware"
	"github.com/aegis-imaging/aegis/api/model"
)

func (s *Server) requireProjectReadAccess(w http.ResponseWriter, r *http.Request, projectID string) (*model.UserProjectAccess, bool) {
	if strings.TrimSpace(projectID) == "" {
		s.writeError(w, http.StatusBadRequest, "project_id is required")
		return nil, false
	}

	user := middleware.UserFromContext(r.Context())
	if user == nil {
		if _, err := model.GetProjectByID(r.Context(), s.db, projectID); err != nil {
			s.writeError(w, http.StatusNotFound, "project not found")
			return nil, false
		}
		return nil, true
	}

	_, access, allowed, err := middleware.ResolveProjectAccess(r.Context(), s.db, user, projectID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.writeError(w, http.StatusNotFound, "project not found")
			return nil, false
		}
		s.writeError(w, http.StatusInternalServerError, "failed to verify project access")
		return nil, false
	}
	if !allowed {
		s.writeError(w, http.StatusNotFound, "project not found")
		return nil, false
	}
	return access, true
}

func (s *Server) requireStudyReadAccessByID(w http.ResponseWriter, r *http.Request, studyID string) (*model.Study, *model.UserProjectAccess, bool) {
	study, err := model.GetStudyByID(r.Context(), s.db, studyID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.writeError(w, http.StatusNotFound, "study not found")
			return nil, nil, false
		}
		s.writeError(w, http.StatusInternalServerError, "failed to get study")
		return nil, nil, false
	}

	access, ok := s.requireProjectReadAccess(w, r, study.ProjectID)
	if !ok {
		return nil, nil, false
	}

	if access != nil && access.IsSiteScoped() {
		if study.InstitutionID == nil || *study.InstitutionID != *access.InstitutionID {
			s.writeError(w, http.StatusNotFound, "study not found")
			return nil, nil, false
		}
	}

	return study, access, true
}

func (s *Server) requireStudyReadAccessByUID(w http.ResponseWriter, r *http.Request, studyUID string) (*model.Study, *model.UserProjectAccess, bool) {
	study, err := model.GetStudyByUID(r.Context(), s.db, studyUID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.writeError(w, http.StatusNotFound, "study not found")
			return nil, nil, false
		}
		s.writeError(w, http.StatusInternalServerError, "failed to get study")
		return nil, nil, false
	}

	access, ok := s.requireProjectReadAccess(w, r, study.ProjectID)
	if !ok {
		return nil, nil, false
	}

	if access != nil && access.IsSiteScoped() {
		if study.InstitutionID == nil || *study.InstitutionID != *access.InstitutionID {
			s.writeError(w, http.StatusNotFound, "study not found")
			return nil, nil, false
		}
	}

	return study, access, true
}

func (s *Server) requireResearcherProjectScope(w http.ResponseWriter, r *http.Request, projectID string) (*model.UserProjectAccess, bool) {
	user := middleware.UserFromContext(r.Context())
	if user == nil || user.Role != "researcher" {
		return nil, true
	}

	if strings.TrimSpace(projectID) == "" {
		s.writeError(w, http.StatusBadRequest, "project_id is required for researcher queries")
		return nil, false
	}

	access, ok := s.requireProjectReadAccess(w, r, projectID)
	if !ok {
		return nil, false
	}
	return access, true
}
