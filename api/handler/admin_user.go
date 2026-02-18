package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/msenjem/aegis/api/model"
)

// ListAdminUsers returns all admin users.
// GET /api/admin-users
func (s *Server) ListAdminUsers(w http.ResponseWriter, r *http.Request) {
	users, err := model.ListAdminUsers(r.Context(), s.db)
	if err != nil {
		log.Printf("list admin users: %v", err)
		s.writeError(w, http.StatusInternalServerError, "failed to list users")
		return
	}
	if users == nil {
		users = []model.AdminUser{}
	}
	s.writeJSON(w, http.StatusOK, users)
}

// CreateAdminUser creates a new admin user.
// POST /api/admin-users
func (s *Server) CreateAdminUser(w http.ResponseWriter, r *http.Request) {
	var u model.AdminUser
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if u.Email == "" {
		s.writeError(w, http.StatusBadRequest, "email is required")
		return
	}
	if u.Role != "admin" && u.Role != "viewer" {
		u.Role = "admin"
	}
	u.Enabled = true

	if err := model.CreateAdminUser(r.Context(), s.db, &u); err != nil {
		log.Printf("create admin user: %v", err)
		s.writeError(w, http.StatusInternalServerError, "failed to create user")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "admin_user.created", "admin",
		"admin_user", u.ID, clientIP(r), map[string]any{
			"email": u.Email,
			"role":  u.Role,
		})
	s.writeJSON(w, http.StatusCreated, u)
}

// UpdateAdminUser updates an existing admin user.
// PUT /api/admin-users/{id}
func (s *Server) UpdateAdminUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	existing, err := model.GetAdminUserByID(r.Context(), s.db, id)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "user not found")
		return
	}

	var u model.AdminUser
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if u.Email == "" {
		s.writeError(w, http.StatusBadRequest, "email is required")
		return
	}
	if u.Role != "admin" && u.Role != "viewer" {
		s.writeError(w, http.StatusBadRequest, "role must be admin or viewer")
		return
	}
	u.ID = existing.ID

	if err := model.UpdateAdminUser(r.Context(), s.db, &u); err != nil {
		log.Printf("update admin user %s: %v", id, err)
		s.writeError(w, http.StatusInternalServerError, "failed to update user")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "admin_user.updated", "admin",
		"admin_user", id, clientIP(r), map[string]any{
			"email":   u.Email,
			"role":    u.Role,
			"enabled": u.Enabled,
		})
	updated, _ := model.GetAdminUserByID(r.Context(), s.db, id)
	s.writeJSON(w, http.StatusOK, updated)
}

// DeleteAdminUser removes an admin user.
// DELETE /api/admin-users/{id}
func (s *Server) DeleteAdminUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := model.GetAdminUserByID(r.Context(), s.db, id); err != nil {
		s.writeError(w, http.StatusNotFound, "user not found")
		return
	}
	if err := model.DeleteAdminUser(r.Context(), s.db, id); err != nil {
		log.Printf("delete admin user %s: %v", id, err)
		s.writeError(w, http.StatusInternalServerError, "failed to delete user")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "admin_user.deleted", "admin",
		"admin_user", id, clientIP(r), nil)
	w.WriteHeader(http.StatusNoContent)
}
