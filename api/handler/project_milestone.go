package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/aegis-imaging/aegis/api/model"
)

// ListProjectMilestones GET /api/projects/{id}/milestones
func (s *Server) ListProjectMilestones(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	milestones, err := model.ListProjectMilestones(r.Context(), s.db, projectID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	if milestones == nil {
		milestones = []model.ProjectMilestone{}
	}
	s.writeJSON(w, http.StatusOK, milestones)
}

// CreateProjectMilestone POST /api/projects/{id}/milestones
func (s *Server) CreateProjectMilestone(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")

	var req struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		s.writeError(w, http.StatusBadRequest, "title is required")
		return
	}
	if len(req.Title) > 200 {
		s.writeError(w, http.StatusBadRequest, "title must be 200 characters or fewer")
		return
	}

	m := &model.ProjectMilestone{
		ProjectID:   projectID,
		Title:       req.Title,
		Description: strings.TrimSpace(req.Description),
	}
	if err := model.CreateProjectMilestone(r.Context(), s.db, m); err != nil {
		s.writeError(w, http.StatusInternalServerError, "create failed")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "project.milestone_created", actorEmail(r), "project", projectID, clientIP(r), map[string]any{
		"title": m.Title,
	})
	s.writeJSON(w, http.StatusCreated, m)
}

// UpdateProjectMilestone PATCH /api/projects/{id}/milestones/{milestoneID}
func (s *Server) UpdateProjectMilestone(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	milestoneID := r.PathValue("milestoneID")

	var req struct {
		Reached *bool `json:"reached"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Reached == nil {
		s.writeError(w, http.StatusBadRequest, "reached is required")
		return
	}

	if err := model.UpdateProjectMilestoneReached(r.Context(), s.db, milestoneID, *req.Reached); err != nil {
		s.writeError(w, http.StatusInternalServerError, "update failed")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "project.milestone_updated", actorEmail(r), "project", projectID, clientIP(r), map[string]any{
		"milestone_id": milestoneID, "reached": *req.Reached,
	})
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// DeleteProjectMilestone DELETE /api/projects/{id}/milestones/{milestoneID}
func (s *Server) DeleteProjectMilestone(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	milestoneID := r.PathValue("milestoneID")
	if err := model.DeleteProjectMilestone(r.Context(), s.db, milestoneID, projectID); err != nil {
		s.writeError(w, http.StatusInternalServerError, "delete failed")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "project.milestone_deleted", actorEmail(r), "project", projectID, clientIP(r), map[string]any{
		"milestone_id": milestoneID,
	})
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
