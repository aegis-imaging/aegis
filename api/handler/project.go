package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/msenjem/aegis/api/model"
)

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

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
	if req.Name == "" {
		s.writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Slug == "" {
		req.Slug = slugRe.ReplaceAllString(strings.ToLower(req.Name), "-")
		req.Slug = strings.Trim(req.Slug, "-")
	}

	project, err := model.CreateProject(r.Context(), s.db, req.Name, req.Slug, req.Description)
	if err != nil {
		s.writeError(w, http.StatusConflict, "project slug already exists")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "project.created", actorEmail(r),
		"project", project.ID, clientIP(r), map[string]any{"name": project.Name, "slug": project.Slug})
	s.writeJSON(w, http.StatusCreated, project)
}

// GetProject returns a single project by ID.
// GET /api/projects/{id}
func (s *Server) GetProject(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	project, err := model.GetProjectByID(r.Context(), s.db, id)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "project not found")
		return
	}
	s.writeJSON(w, http.StatusOK, project)
}

// UpdateProject updates a project's name, slug, and description.
// PUT /api/projects/{id}
func (s *Server) UpdateProject(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := model.GetProjectByID(r.Context(), s.db, id); err != nil {
		s.writeError(w, http.StatusNotFound, "project not found")
		return
	}

	var req createProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Name == "" {
		s.writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Slug == "" {
		req.Slug = slugRe.ReplaceAllString(strings.ToLower(req.Name), "-")
		req.Slug = strings.Trim(req.Slug, "-")
	}

	project, err := model.UpdateProject(r.Context(), s.db, id, req.Name, req.Slug, req.Description)
	if err != nil {
		log.Printf("update project %s: %v", id, err)
		s.writeError(w, http.StatusConflict, "slug already exists or update failed")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "project.updated", actorEmail(r),
		"project", id, clientIP(r), map[string]any{"name": project.Name, "slug": project.Slug})
	s.writeJSON(w, http.StatusOK, project)
}
