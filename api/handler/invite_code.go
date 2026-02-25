package handler

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/aegis-imaging/aegis/api/email"
	"github.com/aegis-imaging/aegis/api/model"
)

// ValidateInviteCode checks whether a landing page invite code is valid.
// Public, rate-limited — no authentication required.
//
// POST /api/invite/validate
func (s *Server) ValidateInviteCode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Code == "" {
		s.writeJSON(w, http.StatusOK, map[string]any{"valid": false})
		return
	}

	valid, err := model.ValidateAndRecordInviteCode(r.Context(), s.db, req.Code, clientIP(r))
	if err != nil {
		log.Printf("invite validate: db error: %v", err)
		// Return invalid rather than exposing internal error to public endpoint.
		s.writeJSON(w, http.StatusOK, map[string]any{"valid": false})
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]any{"valid": valid})
}

// ListInviteCodes returns all invite codes. Admin-only.
//
// GET /api/invite-codes
func (s *Server) ListInviteCodes(w http.ResponseWriter, r *http.Request) {
	codes, err := model.ListInviteCodes(r.Context(), s.db)
	if err != nil {
		log.Printf("list invite codes: %v", err)
		s.writeError(w, http.StatusInternalServerError, "failed to list invite codes")
		return
	}
	if codes == nil {
		codes = []model.InviteCode{}
	}
	s.writeJSON(w, http.StatusOK, map[string]any{
		"codes": codes,
		"total": len(codes),
	})
}

// CreateInviteCode generates a new unique invite code. Admin-only.
//
// POST /api/invite-codes
func (s *Server) CreateInviteCode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Label string `json:"label"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	ic, err := model.CreateInviteCode(r.Context(), s.db, req.Label)
	if err != nil {
		log.Printf("create invite code: %v", err)
		s.writeError(w, http.StatusInternalServerError, "failed to create invite code")
		return
	}

	model.CreateAuditEntry(r.Context(), s.db, "invite_code.created", actorEmail(r), "invite_code", ic.ID, clientIP(r),
		map[string]any{"label": ic.Label})

	s.writeJSON(w, http.StatusCreated, ic)
}

// RevokeInviteCode disables an invite code so it can no longer be used. Admin-only.
//
// POST /api/invite-codes/{id}/revoke
func (s *Server) RevokeInviteCode(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := model.RevokeInviteCode(r.Context(), s.db, id); err != nil {
		if err == sql.ErrNoRows {
			s.writeError(w, http.StatusNotFound, "invite code not found")
			return
		}
		log.Printf("revoke invite code %s: %v", id, err)
		s.writeError(w, http.StatusInternalServerError, "failed to revoke invite code")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "invite_code.revoked", actorEmail(r), "invite_code", id, clientIP(r), nil)
	s.writeJSON(w, http.StatusOK, map[string]any{"status": "revoked"})
}

// SendInviteCode emails an invite code to a specified recipient. Admin-only.
//
// POST /api/invite-codes/{id}/send
func (s *Server) SendInviteCode(w http.ResponseWriter, r *http.Request) {
	if s.cfg.SMTPHost == "" {
		s.writeError(w, http.StatusServiceUnavailable,
			"email is not configured on this server (set SMTP_HOST)")
		return
	}

	id := r.PathValue("id")
	ic, err := model.GetInviteCode(r.Context(), s.db, id)
	if err != nil {
		if err == sql.ErrNoRows {
			s.writeError(w, http.StatusNotFound, "invite code not found")
			return
		}
		log.Printf("send invite code %s: db: %v", id, err)
		s.writeError(w, http.StatusInternalServerError, "failed to look up invite code")
		return
	}
	if !ic.Enabled {
		s.writeError(w, http.StatusConflict, "invite code is revoked")
		return
	}

	var req struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	req.Email = strings.TrimSpace(req.Email)
	req.Name = strings.TrimSpace(req.Name)
	if req.Email == "" || !strings.Contains(req.Email, "@") {
		s.writeError(w, http.StatusBadRequest, "valid email is required")
		return
	}
	if req.Name == "" {
		req.Name = ic.Label
	}

	inviteURL := s.cfg.LandingBaseURL + "/?invite=" + ic.Code
	subject, body := email.InviteCodeIssued(req.Name, ic.Code, inviteURL, s.cfg.LandingBaseURL)
	if err := s.mailer.Send(r.Context(), req.Email, subject, body); err != nil {
		log.Printf("send invite code %s to %s: %v", id, req.Email, err)
		s.writeError(w, http.StatusInternalServerError, "failed to send email")
		return
	}

	model.CreateAuditEntry(r.Context(), s.db, "invite_code.sent", actorEmail(r),
		"invite_code", id, clientIP(r),
		map[string]any{"to": req.Email, "name": req.Name, "code": ic.Code})

	s.writeJSON(w, http.StatusOK, map[string]any{"status": "sent"})
}

// GetInviteCodeActivity returns the audit trail for the user associated with an invite code.
// The user email is resolved via the linked invite_request record.
//
// GET /api/invite-codes/{id}/activity
func (s *Server) GetInviteCodeActivity(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	userEmail, err := model.GetInviteCodeUserEmail(r.Context(), s.db, id)
	if err != nil {
		log.Printf("get invite code activity %s: %v", id, err)
		s.writeError(w, http.StatusInternalServerError, "failed to look up invite code")
		return
	}
	if userEmail == nil {
		s.writeError(w, http.StatusNotFound, "no email linked to this invite code")
		return
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}

	entries, err := model.ListAuditEntriesByActor(r.Context(), s.db, *userEmail, limit)
	if err != nil {
		log.Printf("get invite code activity %s: audit query: %v", id, err)
		s.writeError(w, http.StatusInternalServerError, "failed to load activity")
		return
	}
	if entries == nil {
		entries = []model.AuditEntry{}
	}
	s.writeJSON(w, http.StatusOK, map[string]any{
		"email":   *userEmail,
		"entries": entries,
		"total":   len(entries),
	})
}

// SendAdminInvite creates an admin_users record for the invite code holder and emails
// them a link to the admin dashboard. Chooseable role (admin|viewer, default viewer).
// Requires the invite code to have a linked invite_request (which carries the email).
//
// POST /api/invite-codes/{id}/send-admin-invite
func (s *Server) SendAdminInvite(w http.ResponseWriter, r *http.Request) {
	if s.cfg.SMTPHost == "" {
		s.writeError(w, http.StatusServiceUnavailable,
			"email is not configured on this server (set SMTP_HOST)")
		return
	}

	id := r.PathValue("id")
	userEmail, err := model.GetInviteCodeUserEmail(r.Context(), s.db, id)
	if err != nil {
		log.Printf("send admin invite %s: %v", id, err)
		s.writeError(w, http.StatusInternalServerError, "failed to look up invite code")
		return
	}
	if userEmail == nil {
		s.writeError(w, http.StatusNotFound,
			"no email linked to this invite code — only codes created from access requests can be used to invite admin users")
		return
	}

	var req struct {
		Role string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Role != "admin" && req.Role != "viewer" {
		req.Role = "viewer"
	}

	// Create the admin_users record. If the email already exists the DB unique
	// constraint fires — surface that as a 409 rather than a 500.
	u := model.AdminUser{
		Email:   *userEmail,
		Role:    req.Role,
		Enabled: true,
	}
	adminCreated := true
	if err := model.CreateAdminUser(r.Context(), s.db, &u); err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			adminCreated = false
			// User already exists — still send the email so they get the dashboard URL,
			// but don't overwrite the existing record.
		} else {
			log.Printf("send admin invite %s: create admin user: %v", id, err)
			s.writeError(w, http.StatusInternalServerError, "failed to create admin user")
			return
		}
	}

	subject, body := email.AdminDashboardInvite(*userEmail, s.cfg.AdminDashboardURL, req.Role)
	if err := s.mailer.Send(r.Context(), *userEmail, subject, body); err != nil {
		log.Printf("send admin invite %s: email: %v", id, err)
		s.writeError(w, http.StatusInternalServerError, "failed to send email")
		return
	}

	if adminCreated {
		model.CreateAuditEntry(r.Context(), s.db, "admin_user.created", actorEmail(r),
			"admin_user", u.ID, clientIP(r), map[string]any{"email": u.Email, "role": u.Role})
	}
	model.CreateAuditEntry(r.Context(), s.db, "invite_code.admin_invite_sent", actorEmail(r),
		"invite_code", id, clientIP(r),
		map[string]any{"to": *userEmail, "role": req.Role, "admin_user_created": adminCreated})

	s.writeJSON(w, http.StatusOK, map[string]any{
		"status":              "sent",
		"email":               *userEmail,
		"role":                req.Role,
		"admin_user_created":  adminCreated,
	})
}

// DeleteInviteCode permanently removes an invite code. Admin-only.
//
// DELETE /api/invite-codes/{id}
func (s *Server) DeleteInviteCode(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := model.DeleteInviteCode(r.Context(), s.db, id); err != nil {
		if err == sql.ErrNoRows {
			s.writeError(w, http.StatusNotFound, "invite code not found")
			return
		}
		log.Printf("delete invite code %s: %v", id, err)
		s.writeError(w, http.StatusInternalServerError, "failed to delete invite code")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "invite_code.deleted", actorEmail(r), "invite_code", id, clientIP(r), nil)
	w.WriteHeader(http.StatusNoContent)
}
