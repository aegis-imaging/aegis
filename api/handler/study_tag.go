package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/aegis-imaging/aegis/api/model"
)

// ListStudyTags GET /api/studies/{id}/tags
func (s *Server) ListStudyTags(w http.ResponseWriter, r *http.Request) {
	studyID := r.PathValue("id")
	tags, err := model.ListStudyTags(r.Context(), s.db, studyID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	if tags == nil {
		tags = []model.StudyTag{}
	}
	s.writeJSON(w, http.StatusOK, tags)
}

// AddStudyTag POST /api/studies/{id}/tags
func (s *Server) AddStudyTag(w http.ResponseWriter, r *http.Request) {
	studyID := r.PathValue("id")

	study, err := model.GetStudyByID(r.Context(), s.db, studyID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "study not found")
		return
	}

	var req struct {
		Tag string `json:"tag"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Tag = strings.TrimSpace(req.Tag)
	if req.Tag == "" {
		s.writeError(w, http.StatusBadRequest, "tag is required")
		return
	}
	if len(req.Tag) > 80 {
		s.writeError(w, http.StatusBadRequest, "tag must be 80 characters or fewer")
		return
	}

	actor := actorEmail(r)
	t, err := model.AddStudyTag(r.Context(), s.db, studyID, study.ProjectID, req.Tag, actor)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "create failed")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "study.tag_added", actor, "study", studyID, clientIP(r), map[string]any{
		"tag": req.Tag,
	})
	s.writeJSON(w, http.StatusCreated, t)
}

// DeleteStudyTag DELETE /api/studies/{id}/tags/{tagID}
func (s *Server) DeleteStudyTag(w http.ResponseWriter, r *http.Request) {
	studyID := r.PathValue("id")
	tagID := r.PathValue("tagID")
	if err := model.DeleteStudyTag(r.Context(), s.db, tagID, studyID); err != nil {
		s.writeError(w, http.StatusInternalServerError, "delete failed")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "study.tag_removed", actorEmail(r), "study", studyID, clientIP(r), map[string]any{
		"tag_id": tagID,
	})
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// GetProjectTagTaxonomy GET /api/projects/{id}/tag-taxonomy
func (s *Server) GetProjectTagTaxonomy(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	tags, err := model.ListProjectTagTaxonomy(r.Context(), s.db, projectID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	if tags == nil {
		tags = []model.TagCount{}
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"tags": tags, "total": len(tags)})
}
