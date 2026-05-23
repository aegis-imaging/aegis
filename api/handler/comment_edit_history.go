package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/aegis-imaging/aegis/api/model"
)

// ListCommentEditHistory GET /api/comments/{id}/history
func (s *Server) ListCommentEditHistory(w http.ResponseWriter, r *http.Request) {
	commentID := r.PathValue("id")
	history, err := model.ListCommentEditHistory(r.Context(), s.db, commentID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	if history == nil {
		history = []model.CommentEditHistory{}
	}
	s.writeJSON(w, http.StatusOK, history)
}

// EditStudyComment PUT /api/comments/{id}
// Updates a comment body and records the edit in history.
func (s *Server) EditStudyComment(w http.ResponseWriter, r *http.Request) {
	commentID := r.PathValue("id")

	var req struct {
		Body string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Body = strings.TrimSpace(req.Body)
	if req.Body == "" {
		s.writeError(w, http.StatusBadRequest, "body is required")
		return
	}
	if len(req.Body) > 5000 {
		s.writeError(w, http.StatusBadRequest, "body must be 5000 characters or fewer")
		return
	}

	// We need to get the old body for history. Use a simple query.
	var oldBody string
	err := s.db.QueryRowContext(r.Context(), `SELECT body FROM study_comments WHERE id = $1`, commentID).Scan(&oldBody)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "comment not found")
		return
	}

	actor := actorEmail(r)

	// Record edit history.
	h := &model.CommentEditHistory{
		CommentID: commentID,
		OldBody:   oldBody,
		NewBody:   req.Body,
		EditedBy:  actor,
	}
	if err := model.CreateCommentEditHistory(r.Context(), s.db, h); err != nil {
		s.writeError(w, http.StatusInternalServerError, "record history failed")
		return
	}

	// Update the comment.
	if err := model.UpdateStudyComment(r.Context(), s.db, commentID, req.Body); err != nil {
		s.writeError(w, http.StatusInternalServerError, "update failed")
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
