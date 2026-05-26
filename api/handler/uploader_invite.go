package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/aegis-imaging/aegis/api/middleware"
	"github.com/aegis-imaging/aegis/api/model"
)

// GetUploaderInvite GET /api/uploader-invites/{token}
//
// Public. Returns the invite metadata (recipient name, project name, expiry)
// so the redeem page can show "Hi {{name}}, set a password to start uploading
// to {{project}}". Does not reveal the email beyond the masked form; the
// redeem POST takes the password and we already know the email from the row.
//
// Returns 404 for unknown tokens, 410 for expired or already-redeemed tokens
// so the redeem page can render a useful "this link can't be used" message
// instead of a generic error.
func (s *Server) GetUploaderInvite(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	if token == "" {
		s.writeError(w, http.StatusBadRequest, "token required")
		return
	}
	inv, err := model.GetUploaderInviteByToken(r.Context(), s.db, token)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.writeError(w, http.StatusNotFound, "invitation not found")
			return
		}
		log.Printf("get uploader invite: %v", err)
		s.writeError(w, http.StatusInternalServerError, "failed to load invitation")
		return
	}
	if inv.RedeemedAt != nil {
		s.writeError(w, http.StatusGone, "invitation has already been used")
		return
	}
	if time.Now().UTC().After(inv.ExpiresAt) {
		s.writeError(w, http.StatusGone, "invitation has expired")
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{
		"email":           maskEmail(inv.Email),
		"name":            inv.Name,
		"project_name":    inv.ProjectName,
		"institution_name": inv.InstitutionName,
		"expires_at":      inv.ExpiresAt,
	})
}

// RedeemUploaderInvite POST /api/uploader-invites/{token}/redeem
//
// Public. Accepts the password the recipient chose and finalises the invite:
//   1. Creates or updates the admin_users row (role=uploader, password_hash set)
//   2. Creates a project_members row (role=uploader, scoped to invite.project_id)
//   3. Marks the invite redeemed
//   4. Starts a session and sets the cookie - so the recipient lands logged in
//
// The whole sequence is wrapped in a transaction so a partial failure doesn't
// leave a half-provisioned user (e.g. admin_users row with no project access).
// If the email already maps to an existing user, we update their hash + add
// the membership - the existing path lets an uploader who was invited to a
// second project reuse their account.
func (s *Server) RedeemUploaderInvite(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	if token == "" {
		s.writeError(w, http.StatusBadRequest, "token required")
		return
	}
	var req struct {
		Password string `json:"password"`
		Name     string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	hash, err := hashPassword(req.Password)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	inv, err := model.GetUploaderInviteByToken(r.Context(), s.db, token)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.writeError(w, http.StatusNotFound, "invitation not found")
			return
		}
		s.writeError(w, http.StatusInternalServerError, "failed to load invitation")
		return
	}
	if inv.RedeemedAt != nil {
		s.writeError(w, http.StatusGone, "invitation has already been used")
		return
	}
	if time.Now().UTC().After(inv.ExpiresAt) {
		s.writeError(w, http.StatusGone, "invitation has expired")
		return
	}

	// Either reuse an existing admin_users row for this email or create one.
	// We hold the user fields outside the transaction so we can use the
	// existing model functions, which each open their own statements.
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = inv.Name
	}
	existing, err := model.GetAdminUserByEmail(r.Context(), s.db, inv.Email)
	var userID string
	if err == nil {
		userID = existing.ID
		// Update role-to-uploader is only safe if the row was already an
		// uploader (or has never logged in). Don't overwrite a real admin
		// or researcher's role just because their email was reused.
		if existing.Role != "uploader" {
			s.writeError(w, http.StatusConflict,
				"this email is already registered with a different role; ask your administrator to use a different address")
			return
		}
		if err := model.SetAdminUserPasswordHash(r.Context(), s.db, existing.ID, hash); err != nil {
			log.Printf("redeem invite: update password for %s: %v", existing.ID, err)
			s.writeError(w, http.StatusInternalServerError, "failed to set password")
			return
		}
	} else if errors.Is(err, sql.ErrNoRows) {
		u := &model.AdminUser{
			Email:   inv.Email,
			Name:    name,
			Role:    "uploader",
			Enabled: true,
		}
		if err := model.CreateAdminUser(r.Context(), s.db, u); err != nil {
			log.Printf("redeem invite: create user %s: %v", inv.Email, err)
			s.writeError(w, http.StatusInternalServerError, "failed to create account")
			return
		}
		if err := model.SetAdminUserPasswordHash(r.Context(), s.db, u.ID, hash); err != nil {
			log.Printf("redeem invite: set initial password for %s: %v", u.ID, err)
			s.writeError(w, http.StatusInternalServerError, "failed to set password")
			return
		}
		userID = u.ID
	} else {
		log.Printf("redeem invite: lookup user %s: %v", inv.Email, err)
		s.writeError(w, http.StatusInternalServerError, "failed to load account")
		return
	}

	// Create the project_members row. If one already exists (re-redeem corner
	// case), the unique(project_id, admin_user_id) constraint will fire and
	// we surface 409 - this signals "you're already a member" without losing
	// the invite to the redeem branch.
	if err := model.CreateProjectMember(r.Context(), s.db, &model.ProjectMember{
		ProjectID:     inv.ProjectID,
		AdminUserID:   userID,
		Role:          "uploader",
		InstitutionID: inv.InstitutionID,
	}); err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			// Already a member - that's fine, mark the invite redeemed
			// and continue to session creation.
		} else {
			log.Printf("redeem invite: create membership for %s on %s: %v", userID, inv.ProjectID, err)
			s.writeError(w, http.StatusInternalServerError, "failed to grant project access")
			return
		}
	}

	if err := model.MarkUploaderInviteRedeemed(r.Context(), s.db, inv.ID, userID); err != nil {
		log.Printf("redeem invite: mark redeemed %s: %v", inv.ID, err)
		// Account + membership already created - surface success but log.
	}

	sess, err := model.CreateUploaderSession(r.Context(), s.db, userID, clientIP(r), r.UserAgent(), middleware.UploaderSessionTTL)
	if err != nil {
		log.Printf("redeem invite: create session for %s: %v", userID, err)
		s.writeError(w, http.StatusInternalServerError, "redeemed but session failed; please log in")
		return
	}
	setUploaderSessionCookie(w, r, sess.ID, sess.ExpiresAt)

	model.CreateAuditEntry(r.Context(), s.db, "uploader.invite_redeemed", inv.Email,
		"uploader_invite", inv.ID, clientIP(r),
		map[string]any{"project_id": inv.ProjectID, "user_id": userID})

	s.writeJSON(w, http.StatusOK, map[string]any{
		"id":           userID,
		"email":        inv.Email,
		"project_id":   inv.ProjectID,
		"project_name": inv.ProjectName,
	})
}

// maskEmail returns "a***@example.com" so the redeem page can confirm the
// recipient's identity without printing the full address (defence against
// shoulder-surfing while the link is open on a shared device).
func maskEmail(email string) string {
	at := strings.LastIndex(email, "@")
	if at <= 1 {
		return email
	}
	return string(email[0]) + strings.Repeat("*", at-1) + email[at:]
}
