package handler

import (
	"net/http"

	"github.com/aegis-imaging/aegis/api/middleware"
)

// actorEmail returns the authenticated user's email from the request context,
// or "anonymous" if no auth context is present (public routes).
func actorEmail(r *http.Request) string {
	if u := middleware.UserFromContext(r.Context()); u != nil {
		return u.Email
	}
	return "anonymous"
}

// AuthMe returns the currently authenticated user's identity.
func (s *Server) AuthMe(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		s.writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]string{
		"id":    user.ID,
		"email": user.Email,
		"name":  user.Name,
		"role":  user.Role,
	})
}
