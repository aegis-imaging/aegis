package handler

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/aegis-imaging/aegis/api/middleware"
	"github.com/aegis-imaging/aegis/api/model"
)

// projectNameForStudy returns the project name for the given project ID.
// Returns an empty string on any error (non-fatal — emails still send without it).
func projectNameForStudy(ctx context.Context, db *sql.DB, projectID string) string {
	if projectID == "" {
		return ""
	}
	p, err := model.GetProjectByID(ctx, db, projectID)
	if err != nil {
		return ""
	}
	return p.Name
}

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
