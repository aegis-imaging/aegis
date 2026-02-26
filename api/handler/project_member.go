package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/aegis-imaging/aegis/api/middleware"
	"github.com/aegis-imaging/aegis/api/model"
)

// canManageProjectMembers returns true when the authenticated user may add, update,
// or remove members for the given project.
//   - Platform admin (admin_users.role='admin') → always allowed
//   - Project owner (project_members.role='owner') → allowed for their project
//   - All other roles (viewer, researcher coordinator/reviewer/site_*) → denied
func (s *Server) canManageProjectMembers(r *http.Request, projectID string) bool {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		return false
	}
	if user.Role == "admin" {
		return true
	}
	if user.Role != "researcher" {
		return false
	}
	// Researcher: must be owner of this project.
	access, err := model.GetUserAccessForProject(r.Context(), s.db, user.ID, projectID)
	if err != nil || access == nil {
		return false
	}
	return access.Role == "owner"
}

// ListProjectMembers returns all members of a project.
// GET /api/projects/{id}/members
func (s *Server) ListProjectMembers(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")

	// Verify the project exists and the caller has read access to it.
	user := middleware.UserFromContext(r.Context())
	if !middleware.IsPlatformAdmin(user) {
		// Researcher: must have any membership in this project.
		access, err := model.GetUserAccessForProject(r.Context(), s.db, user.ID, projectID)
		if err != nil || access == nil {
			s.writeError(w, http.StatusNotFound, "project not found")
			return
		}
	}

	members, err := model.ListProjectMembers(r.Context(), s.db, projectID)
	if err != nil {
		log.Printf("list project members: %v", err)
		s.writeError(w, http.StatusInternalServerError, "failed to list project members")
		return
	}
	if members == nil {
		members = []model.ProjectMember{}
	}
	s.writeJSON(w, http.StatusOK, map[string]any{
		"members": members,
		"total":   len(members),
	})
}

// AddProjectMember adds a user to a project with a specified role.
// POST /api/projects/{id}/members
func (s *Server) AddProjectMember(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")

	if !s.canManageProjectMembers(r, projectID) {
		s.writeError(w, http.StatusForbidden, "only project owners and platform admins can manage project members")
		return
	}

	var req struct {
		AdminUserID   string  `json:"admin_user_id"`
		Role          string  `json:"role"`
		InstitutionID *string `json:"institution_id"`
		Notes         string  `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.AdminUserID == "" {
		s.writeError(w, http.StatusBadRequest, "admin_user_id is required")
		return
	}
	if !model.ValidProjectMemberRole(req.Role) {
		s.writeError(w, http.StatusBadRequest, "role must be one of: owner, coordinator, reviewer, site_coordinator, site_viewer")
		return
	}
	// Site roles require an institution_id.
	if (req.Role == "site_coordinator" || req.Role == "site_viewer") && (req.InstitutionID == nil || *req.InstitutionID == "") {
		s.writeError(w, http.StatusBadRequest, "institution_id is required for site roles (site_coordinator, site_viewer)")
		return
	}
	// Coordinating center roles must not have an institution_id.
	if (req.Role == "owner" || req.Role == "coordinator" || req.Role == "reviewer") && req.InstitutionID != nil && *req.InstitutionID != "" {
		s.writeError(w, http.StatusBadRequest, "institution_id must be empty for coordinating center roles (owner, coordinator, reviewer)")
		return
	}

	// Validate that the target user exists.
	targetUser, err := model.GetAdminUserByID(r.Context(), s.db, req.AdminUserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.writeError(w, http.StatusBadRequest, "user not found")
			return
		}
		log.Printf("add project member: get user: %v", err)
		s.writeError(w, http.StatusInternalServerError, "failed to validate user")
		return
	}
	if !targetUser.Enabled {
		s.writeError(w, http.StatusBadRequest, "user is disabled")
		return
	}

	// Validate institution exists (for site roles).
	if req.InstitutionID != nil && *req.InstitutionID != "" {
		if _, err := model.GetInstitutionByID(r.Context(), s.db, *req.InstitutionID); err != nil {
			s.writeError(w, http.StatusBadRequest, "institution not found")
			return
		}
	}

	// Normalise institution_id: empty string → nil
	instID := req.InstitutionID
	if instID != nil && *instID == "" {
		instID = nil
	}

	m := &model.ProjectMember{
		ProjectID:     projectID,
		AdminUserID:   req.AdminUserID,
		Role:          req.Role,
		InstitutionID: instID,
		Notes:         req.Notes,
	}
	if err := model.CreateProjectMember(r.Context(), s.db, m); err != nil {
		log.Printf("add project member: %v", err)
		s.writeError(w, http.StatusInternalServerError, "failed to add project member")
		return
	}

	model.CreateAuditEntry(r.Context(), s.db, "project_member.added", actorEmail(r), "project", projectID, clientIP(r), map[string]any{
		"member_id":      m.ID,
		"user_email":     targetUser.Email,
		"role":           req.Role,
		"institution_id": req.InstitutionID,
	})

	// Reload with joined fields.
	full, err := model.GetProjectMember(r.Context(), s.db, m.ID)
	if err != nil {
		// Not fatal — return what we have.
		s.writeJSON(w, http.StatusCreated, m)
		return
	}
	s.writeJSON(w, http.StatusCreated, full)
}

// UpdateProjectMember changes the role, institution, or notes of an existing member.
// PUT /api/projects/{id}/members/{memberID}
func (s *Server) UpdateProjectMember(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	memberID := r.PathValue("memberID")

	if !s.canManageProjectMembers(r, projectID) {
		s.writeError(w, http.StatusForbidden, "only project owners and platform admins can manage project members")
		return
	}

	existing, err := model.GetProjectMember(r.Context(), s.db, memberID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "project member not found")
		return
	}
	if existing.ProjectID != projectID {
		s.writeError(w, http.StatusNotFound, "project member not found")
		return
	}

	var req struct {
		Role          string  `json:"role"`
		InstitutionID *string `json:"institution_id"`
		Notes         string  `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Role == "" {
		req.Role = existing.Role
	}
	if !model.ValidProjectMemberRole(req.Role) {
		s.writeError(w, http.StatusBadRequest, "role must be one of: owner, coordinator, reviewer, site_coordinator, site_viewer")
		return
	}
	if (req.Role == "site_coordinator" || req.Role == "site_viewer") && (req.InstitutionID == nil || *req.InstitutionID == "") {
		s.writeError(w, http.StatusBadRequest, "institution_id is required for site roles")
		return
	}
	if (req.Role == "owner" || req.Role == "coordinator" || req.Role == "reviewer") && req.InstitutionID != nil && *req.InstitutionID != "" {
		s.writeError(w, http.StatusBadRequest, "institution_id must be empty for coordinating center roles")
		return
	}

	instID := req.InstitutionID
	if instID != nil && *instID == "" {
		instID = nil
	}
	if instID != nil {
		if _, err := model.GetInstitutionByID(r.Context(), s.db, *instID); err != nil {
			s.writeError(w, http.StatusBadRequest, "institution not found")
			return
		}
	}

	existing.Role = req.Role
	existing.InstitutionID = instID
	existing.Notes = req.Notes
	if err := model.UpdateProjectMember(r.Context(), s.db, existing); err != nil {
		log.Printf("update project member: %v", err)
		s.writeError(w, http.StatusInternalServerError, "failed to update project member")
		return
	}

	model.CreateAuditEntry(r.Context(), s.db, "project_member.updated", actorEmail(r), "project", projectID, clientIP(r), map[string]any{
		"member_id":      memberID,
		"user_email":     existing.UserEmail,
		"role":           req.Role,
		"institution_id": req.InstitutionID,
	})

	// Reload with joined fields.
	full, err := model.GetProjectMember(r.Context(), s.db, memberID)
	if err != nil {
		s.writeJSON(w, http.StatusOK, existing)
		return
	}
	s.writeJSON(w, http.StatusOK, full)
}

// RemoveProjectMember removes a user from a project.
// DELETE /api/projects/{id}/members/{memberID}
func (s *Server) RemoveProjectMember(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	memberID := r.PathValue("memberID")

	if !s.canManageProjectMembers(r, projectID) {
		s.writeError(w, http.StatusForbidden, "only project owners and platform admins can manage project members")
		return
	}

	existing, err := model.GetProjectMember(r.Context(), s.db, memberID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "project member not found")
		return
	}
	if existing.ProjectID != projectID {
		s.writeError(w, http.StatusNotFound, "project member not found")
		return
	}

	if err := model.DeleteProjectMember(r.Context(), s.db, memberID); err != nil {
		log.Printf("remove project member: %v", err)
		s.writeError(w, http.StatusInternalServerError, "failed to remove project member")
		return
	}

	model.CreateAuditEntry(r.Context(), s.db, "project_member.removed", actorEmail(r), "project", projectID, clientIP(r), map[string]any{
		"member_id":  memberID,
		"user_email": existing.UserEmail,
		"role":       existing.Role,
	})

	w.WriteHeader(http.StatusNoContent)
}
