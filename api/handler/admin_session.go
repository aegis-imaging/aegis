package handler

import (
	"net/http"
	"strconv"

	"github.com/aegis-imaging/aegis/api/middleware"
	"github.com/aegis-imaging/aegis/api/model"
)

// RecordSession POST /api/auth/session
func (s *Server) RecordSession(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		s.writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	ip := clientIP(r)
	ua := r.UserAgent()
	sess, err := model.RecordAdminSession(r.Context(), s.db, user.ID, ip, ua)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "record failed")
		return
	}
	s.writeJSON(w, http.StatusCreated, sess)
}

// ListSessions GET /api/admin-users/{id}/sessions
func (s *Server) ListSessions(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("id")
	limit := 50
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 {
		limit = l
	}
	sessions, err := model.ListAdminSessions(r.Context(), s.db, userID, limit)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	if sessions == nil {
		sessions = []model.AdminSession{}
	}
	s.writeJSON(w, http.StatusOK, sessions)
}
