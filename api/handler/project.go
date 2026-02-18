package handler

import (
	"encoding/json"
	"net/http"

	"github.com/msenjem/aegis/api/model"
)

func (s *Server) ListProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := model.ListProjects(r.Context(), s.db)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to list projects")
		return
	}
	if projects == nil {
		projects = []model.Project{}
	}
	s.writeJSON(w, http.StatusOK, projects)
}

type createProjectRequest struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
}

func (s *Server) CreateProject(w http.ResponseWriter, r *http.Request) {
	var req createProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Name == "" || req.Slug == "" {
		s.writeError(w, http.StatusBadRequest, "name and slug are required")
		return
	}

	project, err := model.CreateProject(r.Context(), s.db, req.Name, req.Slug, req.Description)
	if err != nil {
		s.writeError(w, http.StatusConflict, "project slug already exists")
		return
	}
	s.writeJSON(w, http.StatusCreated, project)
}
