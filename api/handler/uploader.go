package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/aegis-imaging/aegis/api/email"
	"github.com/aegis-imaging/aegis/api/middleware"
	"github.com/aegis-imaging/aegis/api/model"
)

// inviteUploaderRequest is the body for POST /api/projects/{id}/uploaders.
type inviteUploaderRequest struct {
	Email          string `json:"email"`
	Name           string `json:"name"`
	InstitutionID  string `json:"institution_id"`
	ExpiryDays     int    `json:"expiry_days"`
}

// InviteProjectUploader POST /api/projects/{id}/uploaders
//
// Admin or project-owner researcher invites an outside contributor to upload
// to this project. Creates a uploader_invites row and emails the recipient a
// redeem link. The token is single-use; clicking the link from the email
// lands them on the upload-portal redeem page where they set a password.
//
// Authorization: caller must be a platform admin OR have role='owner' on
// this specific project (matches the per-project setting page where the
// "Uploaders" section will be rendered in PR 3).
func (s *Server) InviteProjectUploader(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	if projectID == "" {
		s.writeError(w, http.StatusBadRequest, "project id required")
		return
	}
	if !s.canManageProjectUploaders(r, projectID) {
		s.writeError(w, http.StatusForbidden, "only project owners and platform admins can manage uploaders")
		return
	}
	if s.cfg.SMTPHost == "" {
		s.writeError(w, http.StatusServiceUnavailable,
			"email is not configured on this server (set SMTP_HOST)")
		return
	}

	proj, err := model.GetProjectByID(r.Context(), s.db, projectID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.writeError(w, http.StatusNotFound, "project not found")
			return
		}
		s.writeError(w, http.StatusInternalServerError, "project lookup failed")
		return
	}

	var req inviteUploaderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.Name = strings.TrimSpace(req.Name)
	if req.Email == "" || !strings.Contains(req.Email, "@") {
		s.writeError(w, http.StatusBadRequest, "valid email required")
		return
	}
	if req.ExpiryDays <= 0 || req.ExpiryDays > 30 {
		// 7 days is enough for a typical "I'll send my data later this week"
		// turnaround; longer windows just sit as latent credentials.
		req.ExpiryDays = 7
	}
	if req.InstitutionID = strings.TrimSpace(req.InstitutionID); req.InstitutionID != "" {
		if _, err := model.GetInstitutionByID(r.Context(), s.db, req.InstitutionID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				s.writeError(w, http.StatusBadRequest, "institution_id does not match a known institution")
				return
			}
			s.writeError(w, http.StatusInternalServerError, "institution lookup failed")
			return
		}
	}

	inv, err := model.CreateUploaderInvite(r.Context(), s.db, model.CreateUploaderInviteInput{
		Email:         req.Email,
		Name:          req.Name,
		ProjectID:     projectID,
		InstitutionID: req.InstitutionID,
		InvitedBy:     actorEmail(r),
		ExpiresAt:     time.Now().UTC().Add(time.Duration(req.ExpiryDays) * 24 * time.Hour),
	})
	if err != nil {
		log.Printf("invite uploader: create row: %v", err)
		s.writeError(w, http.StatusInternalServerError, "failed to create invitation")
		return
	}

	// Prefer the dedicated upload-portal URL when configured — the portal
	// serves the redeem page at /invite/{token}. Fall back to the legacy
	// landing-page path (/upload/invite/{token}) so links keep resolving on
	// deployments that haven't set UPLOAD_PORTAL_BASE_URL yet; the portal
	// accepts both path shapes.
	redeemURL := strings.TrimRight(s.cfg.LandingBaseURL, "/") + "/upload/invite/" + inv.InviteToken
	if base := strings.TrimRight(s.cfg.UploadPortalBaseURL, "/"); base != "" {
		redeemURL = base + "/invite/" + inv.InviteToken
	}
	subject, body := email.UploaderInvite(req.Name, proj.Name, redeemURL, inv.ExpiresAt)
	if err := s.mailer.Send(r.Context(), req.Email, subject, body); err != nil {
		log.Printf("invite uploader: send email to %s: %v", req.Email, err)
		// Don't fail the request - the admin can resend. Surface the invite
		// row so the UI can show a "resend" affordance.
		s.writeJSON(w, http.StatusOK, map[string]any{
			"invite":     inv,
			"redeem_url": redeemURL,
			"email_sent": false,
		})
		return
	}

	model.CreateAuditEntry(r.Context(), s.db, "project.uploader_invited", actorEmail(r),
		"project", projectID, clientIP(r),
		map[string]any{"to": req.Email, "expiry_days": req.ExpiryDays})

	s.writeJSON(w, http.StatusCreated, map[string]any{
		"invite":     inv,
		"redeem_url": redeemURL,
		"email_sent": true,
	})
}

// ListProjectUploaders GET /api/projects/{id}/uploaders
//
// Returns both active members (role=uploader) and pending (un-redeemed,
// un-expired) invites for a project, so the admin UI can render one
// combined table.
func (s *Server) ListProjectUploaders(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	if projectID == "" {
		s.writeError(w, http.StatusBadRequest, "project id required")
		return
	}
	if !s.canManageProjectUploaders(r, projectID) {
		s.writeError(w, http.StatusForbidden, "only project owners and platform admins can view uploaders")
		return
	}
	members, err := model.ListProjectMembers(r.Context(), s.db, projectID)
	if err != nil {
		log.Printf("list project uploaders %s: %v", projectID, err)
		s.writeError(w, http.StatusInternalServerError, "failed to list members")
		return
	}
	uploaders := make([]model.ProjectMember, 0)
	for _, m := range members {
		if m.Role == "uploader" {
			uploaders = append(uploaders, m)
		}
	}

	pending, err := model.ListUploaderInvitesByProject(r.Context(), s.db, projectID)
	if err != nil {
		log.Printf("list project uploader invites %s: %v", projectID, err)
		s.writeError(w, http.StatusInternalServerError, "failed to list invites")
		return
	}
	if pending == nil {
		pending = []model.UploaderInvite{}
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"uploaders":       uploaders,
		"pending_invites": pending,
	})
}

// RevokeProjectUploader DELETE /api/projects/{id}/uploaders/{userId}
//
// Removes the project_members row for the uploader, deletes all their active
// sessions (so any in-flight upload portal tab loses access immediately), and
// records an audit entry. The admin_users row itself is not deleted - they
// may be invited to other projects, and history needs to dereference them.
func (s *Server) RevokeProjectUploader(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	userID := r.PathValue("userId")
	if projectID == "" || userID == "" {
		s.writeError(w, http.StatusBadRequest, "project id and user id required")
		return
	}
	if !s.canManageProjectUploaders(r, projectID) {
		s.writeError(w, http.StatusForbidden, "only project owners and platform admins can revoke uploaders")
		return
	}
	member, err := model.GetProjectMemberByUser(r.Context(), s.db, projectID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.writeError(w, http.StatusNotFound, "uploader not found on this project")
			return
		}
		s.writeError(w, http.StatusInternalServerError, "lookup failed")
		return
	}
	if member.Role != "uploader" {
		// Don't accidentally let a project owner delete a coordinator via
		// the uploader-revoke path; route them to the regular member CRUD.
		s.writeError(w, http.StatusConflict,
			"this user is not an uploader - use project member management to remove other roles")
		return
	}
	if err := model.DeleteProjectMember(r.Context(), s.db, member.ID); err != nil {
		log.Printf("revoke uploader %s on %s: %v", userID, projectID, err)
		s.writeError(w, http.StatusInternalServerError, "failed to revoke access")
		return
	}
	// Best-effort - kill their sessions so an open upload-portal tab can't
	// keep uploading after revocation. Only matters when the uploader has
	// no other project memberships; with multi-project access we'd need a
	// per-project session model.
	other, err := model.GetUserProjectAccess(r.Context(), s.db, userID)
	if err == nil && len(other) == 0 {
		if err := model.DeleteUploaderSessionsByUser(r.Context(), s.db, userID); err != nil {
			log.Printf("revoke uploader %s sessions: %v", userID, err)
		}
	}
	model.CreateAuditEntry(r.Context(), s.db, "project.uploader_revoked", actorEmail(r),
		"project", projectID, clientIP(r),
		map[string]any{"user_id": userID, "user_email": member.UserEmail})

	w.WriteHeader(http.StatusNoContent)
}

// canManageProjectUploaders reports whether the authenticated user has rights
// to invite/list/revoke uploaders on the given project. Platform admins always
// can; researchers must be an 'owner' member of this specific project.
func (s *Server) canManageProjectUploaders(r *http.Request, projectID string) bool {
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
	access, err := model.GetUserAccessForProject(r.Context(), s.db, user.ID, projectID)
	if err != nil || access == nil {
		return false
	}
	return access.Role == "owner"
}
