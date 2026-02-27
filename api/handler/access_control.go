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

type projectWriteIntent string

const (
	projectWriteIntentManageProject projectWriteIntent = "manage_project"
	projectWriteIntentStudyMutation projectWriteIntent = "study_mutation"
	projectWriteIntentApproveReject projectWriteIntent = "approve_reject"
	projectWriteIntentSiteOperation projectWriteIntent = "site_operation"
)

func roleAllowsWriteIntent(access *model.UserProjectAccess, intent projectWriteIntent) bool {
	if access == nil {
		return false
	}

	switch intent {
	case projectWriteIntentManageProject:
		return access.Role == "owner"
	case projectWriteIntentStudyMutation:
		return access.Role == "owner" || access.Role == "coordinator" || access.Role == "reviewer" || access.Role == "site_coordinator"
	case projectWriteIntentApproveReject:
		return access.Role == "owner" || access.Role == "coordinator" || access.Role == "reviewer"
	case projectWriteIntentSiteOperation:
		return access.Role == "owner" || access.Role == "coordinator" || access.Role == "reviewer" || access.Role == "site_coordinator"
	default:
		return false
	}
}

func (s *Server) requireProjectWriteAccess(w http.ResponseWriter, r *http.Request, projectID string, intent projectWriteIntent) (*model.UserProjectAccess, bool) {
	if strings.TrimSpace(projectID) == "" {
		s.writeError(w, http.StatusBadRequest, "project_id is required")
		return nil, false
	}

	user := middleware.UserFromContext(r.Context())
	if user == nil {
		s.writeError(w, http.StatusUnauthorized, "missing authentication context")
		return nil, false
	}

	if user.Role == "admin" {
		return nil, true
	}
	if user.Role != "researcher" {
		s.writeError(w, http.StatusForbidden, "insufficient permissions for this project action")
		return nil, false
	}

	access, ok := s.requireProjectReadAccess(w, r, projectID)
	if !ok {
		return nil, false
	}
	if !roleAllowsWriteIntent(access, intent) {
		s.writeError(w, http.StatusForbidden, "insufficient permissions for this project action")
		return nil, false
	}
	return access, true
}

func (s *Server) requireStudyWriteAccessByID(w http.ResponseWriter, r *http.Request, studyID string, intent projectWriteIntent) (*model.Study, *model.UserProjectAccess, bool) {
	study, err := model.GetStudyByID(r.Context(), s.db, studyID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.writeError(w, http.StatusNotFound, "study not found")
			return nil, nil, false
		}
		s.writeError(w, http.StatusInternalServerError, "failed to get study")
		return nil, nil, false
	}

	access, ok := s.requireProjectWriteAccess(w, r, study.ProjectID, intent)
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

func (s *Server) requireStudyWriteAccessByUID(w http.ResponseWriter, r *http.Request, studyUID string, intent projectWriteIntent) (*model.Study, *model.UserProjectAccess, bool) {
	study, err := model.GetStudyByUID(r.Context(), s.db, studyUID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.writeError(w, http.StatusNotFound, "study not found")
			return nil, nil, false
		}
		s.writeError(w, http.StatusInternalServerError, "failed to get study")
		return nil, nil, false
	}

	access, ok := s.requireProjectWriteAccess(w, r, study.ProjectID, intent)
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
