package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/aegis-imaging/aegis/api/model"
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

// SetProjectRetention sets or clears the retention policy for a project.
// PUT /api/projects/{id}/retention
// Body: {"retention_days": 90} or {"retention_days": null} to clear.
func (s *Server) SetProjectRetention(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := model.GetProjectByID(r.Context(), s.db, id); err != nil {
		s.writeError(w, http.StatusNotFound, "project not found")
		return
	}

	var body struct {
		RetentionDays *int `json:"retention_days"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if body.RetentionDays != nil && *body.RetentionDays <= 0 {
		s.writeError(w, http.StatusBadRequest, "retention_days must be a positive integer or null")
		return
	}

	if err := model.UpdateProjectRetentionDays(r.Context(), s.db, id, body.RetentionDays); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to update retention policy")
		return
	}
	detail := map[string]any{"retention_days": body.RetentionDays}
	model.CreateAuditEntry(r.Context(), s.db, "project.retention_updated", actorEmail(r),
		"project", id, clientIP(r), detail)

	project, _ := model.GetProjectByID(r.Context(), s.db, id)
	s.writeJSON(w, http.StatusOK, project)
}

// ArchiveProject marks a project as archived.
// POST /api/projects/{id}/archive
func (s *Server) ArchiveProject(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := model.GetProjectByID(r.Context(), s.db, id); err != nil {
		s.writeError(w, http.StatusNotFound, "project not found")
		return
	}
	if err := model.ArchiveProject(r.Context(), s.db, id); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to archive project")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "project.archived", actorEmail(r), "project", id, clientIP(r), nil)
	project, _ := model.GetProjectByID(r.Context(), s.db, id)
	s.writeJSON(w, http.StatusOK, project)
}

// RestoreProject unarchives a project.
// POST /api/projects/{id}/restore
func (s *Server) RestoreProject(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := model.GetProjectByID(r.Context(), s.db, id); err != nil {
		s.writeError(w, http.StatusNotFound, "project not found")
		return
	}
	if err := model.RestoreProject(r.Context(), s.db, id); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to restore project")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "project.restored", actorEmail(r), "project", id, clientIP(r), nil)
	project, _ := model.GetProjectByID(r.Context(), s.db, id)
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
