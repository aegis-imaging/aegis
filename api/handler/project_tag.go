package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/aegis-imaging/aegis/api/model"
)

type addProjectTagRequest struct {
	Tag string `json:"tag"`
}

// ListProjectTags GET /api/projects/{id}/tags
func (s *Server) ListProjectTags(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	tags, err := model.ListProjectTags(r.Context(), s.db, projectID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	if tags == nil {
		tags = []model.ProjectTag{}
	}
	s.writeJSON(w, http.StatusOK, tags)
}

// AddProjectTag POST /api/projects/{id}/tags
func (s *Server) AddProjectTag(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")

	var req addProjectTagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	tag := strings.TrimSpace(req.Tag)
	if tag == "" {
		s.writeError(w, http.StatusBadRequest, "tag is required")
		return
	}
	if len(tag) > 80 {
		s.writeError(w, http.StatusBadRequest, "tag must be 80 characters or fewer")
		return
	}

	t, err := model.AddProjectTag(r.Context(), s.db, projectID, tag)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "create failed")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "project.tag_added", actorEmail(r), "project", projectID, clientIP(r), map[string]any{
		"tag": tag,
	})
	s.writeJSON(w, http.StatusCreated, t)
}

// DeleteProjectTag DELETE /api/projects/{id}/tags/{tagID}
func (s *Server) DeleteProjectTag(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	tagID := r.PathValue("tagID")
	if err := model.DeleteProjectTag(r.Context(), s.db, tagID, projectID); err != nil {
		s.writeError(w, http.StatusInternalServerError, "delete failed")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "project.tag_removed", actorEmail(r), "project", projectID, clientIP(r), map[string]any{
		"tag_id": tagID,
	})
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
