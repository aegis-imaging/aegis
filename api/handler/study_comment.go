package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/aegis-imaging/aegis/api/model"
)

type createCommentRequest struct {
	Body     string  `json:"body"`
	ParentID *string `json:"parent_id,omitempty"`
}

// ListStudyComments GET /api/studies/{id}/comments
func (s *Server) ListStudyComments(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	comments, err := model.ListStudyComments(r.Context(), s.db, id)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	if comments == nil {
		comments = []model.StudyComment{}
	}
	s.writeJSON(w, http.StatusOK, comments)
}

// CreateStudyComment POST /api/studies/{id}/comments
func (s *Server) CreateStudyComment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if _, err := model.GetStudyByID(r.Context(), s.db, id); err != nil {
		s.writeError(w, http.StatusNotFound, "study not found")
		return
	}

	var req createCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	body := strings.TrimSpace(req.Body)
	if body == "" {
		s.writeError(w, http.StatusBadRequest, "body is required")
		return
	}
	if len(body) > 5000 {
		s.writeError(w, http.StatusBadRequest, "body must be 5000 characters or fewer")
		return
	}

	c := &model.StudyComment{
		StudyID:  id,
		Author:   actorEmail(r),
		Body:     body,
		ParentID: req.ParentID,
	}
	if err := model.CreateStudyComment(r.Context(), s.db, c); err != nil {
		s.writeError(w, http.StatusInternalServerError, "create failed")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "study.comment_added", actorEmail(r), "study", id, clientIP(r), map[string]any{
		"comment_id": c.ID,
	})
	s.writeJSON(w, http.StatusCreated, c)
}

// DeleteStudyComment DELETE /api/studies/{id}/comments/{commentID}
func (s *Server) DeleteStudyComment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	commentID := r.PathValue("commentID")
	if err := model.DeleteStudyComment(r.Context(), s.db, commentID, id); err != nil {
		s.writeError(w, http.StatusInternalServerError, "delete failed")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "study.comment_removed", actorEmail(r), "study", id, clientIP(r), map[string]any{
		"comment_id": commentID,
	})
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
