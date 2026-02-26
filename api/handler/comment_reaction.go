package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/aegis-imaging/aegis/api/middleware"
	"github.com/aegis-imaging/aegis/api/model"
)

// ListCommentReactions GET /api/comments/{id}/reactions
func (s *Server) ListCommentReactions(w http.ResponseWriter, r *http.Request) {
	commentID := r.PathValue("id")
	reactions, err := model.ListCommentReactions(r.Context(), s.db, commentID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	if reactions == nil {
		reactions = []model.CommentReaction{}
	}
	s.writeJSON(w, http.StatusOK, reactions)
}

// AddCommentReaction POST /api/comments/{id}/reactions
func (s *Server) AddCommentReaction(w http.ResponseWriter, r *http.Request) {
	commentID := r.PathValue("id")

	user := middleware.UserFromContext(r.Context())
	userID := ""
	if user != nil {
		userID = user.ID
	}
	if userID == "" {
		s.writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var req struct {
		Emoji string `json:"emoji"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Emoji = strings.TrimSpace(req.Emoji)
	if req.Emoji == "" {
		s.writeError(w, http.StatusBadRequest, "emoji is required")
		return
	}
	if len(req.Emoji) > 20 {
		s.writeError(w, http.StatusBadRequest, "emoji must be 20 characters or fewer")
		return
	}

	reaction, err := model.AddCommentReaction(r.Context(), s.db, commentID, userID, req.Emoji)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "create failed")
		return
	}
	s.writeJSON(w, http.StatusCreated, reaction)
}

// DeleteCommentReaction DELETE /api/comments/{id}/reactions
func (s *Server) DeleteCommentReaction(w http.ResponseWriter, r *http.Request) {
	commentID := r.PathValue("id")

	user := middleware.UserFromContext(r.Context())
	userID := ""
	if user != nil {
		userID = user.ID
	}
	if userID == "" {
		s.writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var req struct {
		Emoji string `json:"emoji"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Emoji = strings.TrimSpace(req.Emoji)
	if req.Emoji == "" {
		s.writeError(w, http.StatusBadRequest, "emoji is required")
		return
	}

	if err := model.DeleteCommentReaction(r.Context(), s.db, commentID, userID, req.Emoji); err != nil {
		s.writeError(w, http.StatusInternalServerError, "delete failed")
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
