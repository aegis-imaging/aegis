package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/aegis-imaging/aegis/api/model"
)

// ListProjectACL GET /api/projects/{id}/acl
func (s *Server) ListProjectACL(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	entries, err := model.ListProjectACL(r.Context(), s.db, projectID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	if entries == nil {
		entries = []model.ProjectACLEntry{}
	}
	s.writeJSON(w, http.StatusOK, entries)
}

// SetProjectACL POST /api/projects/{id}/acl
func (s *Server) SetProjectACL(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")

	var req struct {
		UserEmail  string `json:"user_email"`
		Permission string `json:"permission"`
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
	req.Permission = strings.TrimSpace(req.Permission)
	validPerms := map[string]bool{"read": true, "write": true, "admin": true}
	if !validPerms[req.Permission] {
		s.writeError(w, http.StatusBadRequest, "permission must be read, write, or admin")
		return
	}

	e := &model.ProjectACLEntry{
		ProjectID:  projectID,
		UserEmail:  req.UserEmail,
		Permission: req.Permission,
		GrantedBy:  actorEmail(r),
	}
	if err := model.UpsertProjectACL(r.Context(), s.db, e); err != nil {
		s.writeError(w, http.StatusInternalServerError, "save failed")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "project.acl_set", actorEmail(r), "project", projectID, clientIP(r), map[string]any{
		"user_email": req.UserEmail, "permission": req.Permission,
	})
	s.writeJSON(w, http.StatusOK, e)
}

// DeleteProjectACL DELETE /api/projects/{id}/acl/{aclID}
func (s *Server) DeleteProjectACL(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	aclID := r.PathValue("aclID")
	if err := model.DeleteProjectACL(r.Context(), s.db, aclID, projectID); err != nil {
		s.writeError(w, http.StatusInternalServerError, "delete failed")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "project.acl_removed", actorEmail(r), "project", projectID, clientIP(r), map[string]any{
		"acl_id": aclID,
	})
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
