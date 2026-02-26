package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/aegis-imaging/aegis/api/model"
)

// ListCommentMentions GET /api/comments/{id}/mentions
func (s *Server) ListCommentMentions(w http.ResponseWriter, r *http.Request) {
	commentID := r.PathValue("id")
	mentions, err := model.ListCommentMentions(r.Context(), s.db, commentID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	if mentions == nil {
		mentions = []model.CommentMention{}
	}
	s.writeJSON(w, http.StatusOK, mentions)
}

// AddCommentMention POST /api/comments/{id}/mentions
func (s *Server) AddCommentMention(w http.ResponseWriter, r *http.Request) {
	commentID := r.PathValue("id")

	var req struct {
		UserEmail string `json:"user_email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.UserEmail = strings.TrimSpace(req.UserEmail)
	if req.UserEmail == "" {
		s.writeError(w, http.StatusBadRequest, "user_email is required")
		return
	}

	m, err := model.CreateCommentMention(r.Context(), s.db, commentID, req.UserEmail)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "create failed")
		return
	}
	s.writeJSON(w, http.StatusCreated, m)
}

// ListMyMentions GET /api/mentions
// Returns recent mentions for the current user.
func (s *Server) ListMyMentions(w http.ResponseWriter, r *http.Request) {
	actor := actorEmail(r)

	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}

	mentions, err := model.ListMentionsForUser(r.Context(), s.db, actor, limit)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	if mentions == nil {
		mentions = []model.CommentMention{}
	}
	s.writeJSON(w, http.StatusOK, mentions)
}
