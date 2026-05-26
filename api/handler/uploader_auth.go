package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/aegis-imaging/aegis/api/middleware"
	"github.com/aegis-imaging/aegis/api/model"
)

// UploaderLoginRequest is the body for POST /api/auth/uploader-login.
type UploaderLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// UploaderLogin POST /api/auth/uploader-login
//
// Public. Validates email+password against admin_users where role='uploader',
// creates a uploader_sessions row, and sets the session id as an HttpOnly,
// SameSite=Lax cookie scoped to the upload portal origin. The cookie name
// (middleware.UploaderSessionCookieName) is distinct from any admin-side
// cookie so the two scopes never mix.
//
// On any failure (unknown email, wrong password, disabled account, wrong
// role) we return the same 401 with a generic message — never disclose
// which factor failed, to keep email enumeration off the table.
func (s *Server) UploaderLogin(w http.ResponseWriter, r *http.Request) {
	var req UploaderLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" || req.Password == "" {
		s.writeError(w, http.StatusBadRequest, "email and password required")
		return
	}

	u, err := model.GetAdminUserByEmail(r.Context(), s.db, req.Email)
	if err != nil {
		// Unknown email - generic failure.
		s.writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if !u.Enabled {
		s.writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if u.Role != "uploader" {
		// Wrong account type - generic failure (don't leak that the email
		// belongs to an admin/researcher).
		s.writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if u.PasswordHash == nil || *u.PasswordHash == "" {
		// Account created but never finished invite redemption.
		s.writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(*u.PasswordHash), []byte(req.Password)); err != nil {
		s.writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	sess, err := model.CreateUploaderSession(r.Context(), s.db, u.ID, clientIP(r), r.UserAgent(), middleware.UploaderSessionTTL)
	if err != nil {
		log.Printf("uploader login: create session for %s: %v", u.ID, err)
		s.writeError(w, http.StatusInternalServerError, "failed to start session")
		return
	}

	setUploaderSessionCookie(w, r, sess.ID, sess.ExpiresAt)

	model.CreateAuditEntry(r.Context(), s.db, "uploader.login", u.Email,
		"admin_user", u.ID, clientIP(r), nil)

	s.writeJSON(w, http.StatusOK, map[string]any{
		"id":    u.ID,
		"email": u.Email,
		"name":  u.Name,
	})
}

// UploaderLogout POST /api/auth/uploader-logout
//
// Invalidates the current session and clears the cookie. Idempotent: returns
// 204 whether or not a session was present. Requires the cookie to identify
// the session row to delete; without it we just clear the client's cookie.
func (s *Server) UploaderLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(middleware.UploaderSessionCookieName); err == nil && c.Value != "" {
		if err := model.DeleteUploaderSession(r.Context(), s.db, c.Value); err != nil {
			log.Printf("uploader logout: delete session: %v", err)
			// Continue - we still want to clear the client cookie.
		}
	}
	clearUploaderSessionCookie(w, r)
	w.WriteHeader(http.StatusNoContent)
}

// UploaderMe GET /api/auth/uploader-me
//
// Returns the current uploader's identity plus the list of projects they
// have upload permission on. The upload portal calls this on load to decide
// whether to show the login screen or the project picker. 401 = not logged
// in, render login screen.
func (s *Server) UploaderMe(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		s.writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	access, err := model.GetUserProjectAccess(r.Context(), s.db, user.ID)
	if err != nil {
		log.Printf("uploader me: load project access for %s: %v", user.ID, err)
		s.writeError(w, http.StatusInternalServerError, "failed to load projects")
		return
	}

	// Hydrate the project ids into name+slug so the picker can render labels.
	type projInfo struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Slug string `json:"slug"`
	}
	out := make([]projInfo, 0, len(access))
	for _, a := range access {
		// Skip non-upload roles. Today only the dedicated 'uploader' role on
		// project_members confers upload rights via this auth path. Admins
		// who happen to also have an uploader account would only see
		// uploader-role memberships here.
		p, err := model.GetProjectByID(r.Context(), s.db, a.ProjectID)
		if err != nil {
			continue
		}
		out = append(out, projInfo{ID: p.ID, Name: p.Name, Slug: p.Slug})
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"id":       user.ID,
		"email":    user.Email,
		"name":     user.Name,
		"projects": out,
	})
}

// setUploaderSessionCookie writes the session cookie with the right security
// flags. Secure is true unless the request came in over plain HTTP (dev).
func setUploaderSessionCookie(w http.ResponseWriter, r *http.Request, sessID string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     middleware.UploaderSessionCookieName,
		Value:    sessID,
		Path:     "/",
		Expires:  expires,
		HttpOnly: true,
		Secure:   isTLSRequest(r),
		SameSite: http.SameSiteLaxMode,
	})
}

func clearUploaderSessionCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     middleware.UploaderSessionCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   isTLSRequest(r),
		SameSite: http.SameSiteLaxMode,
	})
}

// isTLSRequest reports whether the inbound request was served over HTTPS,
// accounting for the X-Forwarded-Proto header set by GCP IAP, Azure App
// Gateway, AWS ALB, and the nginx sidecar. Used to gate the Secure cookie
// flag - on http://localhost during dev, Secure would prevent the cookie
// from being sent back.
func isTLSRequest(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	if xfp := r.Header.Get("X-Forwarded-Proto"); xfp != "" {
		return strings.EqualFold(xfp, "https")
	}
	return false
}

// hashPassword wraps bcrypt with a fixed cost. Centralised so PR 2's
// password-reset flow uses the same cost factor as initial redemption.
func hashPassword(plain string) (string, error) {
	if len(plain) < 8 {
		return "", errors.New("password must be at least 8 characters")
	}
	if len(plain) > 256 {
		// bcrypt itself caps input at 72 bytes; reject anything wild here so
		// we don't silently truncate user-provided passphrases past the limit.
		return "", errors.New("password too long")
	}
	h, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(h), nil
}

// loadProjectMemberRoleForUploader fetches the project_members.role for a
// (project, admin_user) pair, returning sql.ErrNoRows when the user has no
// membership. Used by upload-time authz to confirm the uploader still has
// rights on the specific target project.
func loadProjectMemberRoleForUploader(r *http.Request, db *sql.DB, projectID, userID string) (string, error) {
	access, err := model.GetUserAccessForProject(r.Context(), db, userID, projectID)
	if err != nil {
		return "", err
	}
	if access == nil {
		return "", sql.ErrNoRows
	}
	return access.Role, nil
}
