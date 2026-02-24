package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/aegis-imaging/aegis/api/email"
	"github.com/aegis-imaging/aegis/api/model"
)

// inviteRequestPayload is the data encoded inside an approval token.
type inviteRequestPayload struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Org   string `json:"org,omitempty"`
	TS    int64  `json:"ts"`
}

// signInviteToken encodes payload as base64url JSON and appends an HMAC-SHA256 signature.
// Format: base64url(payload) + "." + base64url(sig)
func signInviteToken(secret string, p inviteRequestPayload) (string, error) {
	raw, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	enc := base64.RawURLEncoding.EncodeToString(raw)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(enc))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return enc + "." + sig, nil
}

// verifyInviteToken validates the HMAC and returns the decoded payload.
// Returns an error if the signature is invalid or the token is older than 7 days.
func verifyInviteToken(secret, token string) (*inviteRequestPayload, error) {
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid token format")
	}
	enc, sigB64 := parts[0], parts[1]

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(enc))
	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(sigB64)) {
		return nil, fmt.Errorf("invalid token signature")
	}

	raw, err := base64.RawURLEncoding.DecodeString(enc)
	if err != nil {
		return nil, fmt.Errorf("decode token: %w", err)
	}
	var p inviteRequestPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, fmt.Errorf("unmarshal token: %w", err)
	}
	if time.Since(time.Unix(p.TS, 0)) > 7*24*time.Hour {
		return nil, fmt.Errorf("token expired")
	}
	return &p, nil
}

// RequestInvite handles POST /api/invite/request — public, rate-limited.
// Collects requester info, sends an approval email to the admin with a one-click link.
func (s *Server) RequestInvite(w http.ResponseWriter, r *http.Request) {
	if s.cfg.InviteRequestSecret == "" {
		// Feature not configured — return success silently so the form still works.
		log.Printf("invite request: INVITE_REQUEST_SECRET not set, dropping request")
		s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}

	var req struct {
		Name    string `json:"name"`
		Email   string `json:"email"`
		Org     string `json:"org"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.TrimSpace(req.Email)
	req.Org = strings.TrimSpace(req.Org)
	req.Message = strings.TrimSpace(req.Message)

	if req.Name == "" {
		s.writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Email == "" || !strings.Contains(req.Email, "@") {
		s.writeError(w, http.StatusBadRequest, "valid email is required")
		return
	}
	if len(req.Message) > 500 {
		s.writeError(w, http.StatusBadRequest, "message too long (max 500 characters)")
		return
	}

	token, err := signInviteToken(s.cfg.InviteRequestSecret, inviteRequestPayload{
		Name:  req.Name,
		Email: req.Email,
		Org:   req.Org,
		TS:    time.Now().Unix(),
	})
	if err != nil {
		log.Printf("invite request: sign token: %v", err)
		s.writeError(w, http.StatusInternalServerError, "failed to process request")
		return
	}

	approvalURL := s.cfg.APIBaseURL + "/api/invite/request/approve?t=" + token
	adminEmail := s.cfg.InviteRequestAdminEmail
	if adminEmail == "" {
		adminEmail = s.cfg.ContactEmail
	}

	subject, body := email.InviteRequest(req.Name, req.Email, req.Org, req.Message, approvalURL)
	if err := s.mailer.Send(r.Context(), adminEmail, subject, body); err != nil {
		log.Printf("invite request: send admin email to %s: %v", adminEmail, err)
		// Still return OK — the form has done its job even if email is misconfigured.
	}

	ir, dbErr := model.CreateInviteRequest(r.Context(), s.db, req.Name, req.Email, req.Org, req.Message, clientIP(r))
	if dbErr != nil {
		log.Printf("invite request: store in db: %v", dbErr)
		// Non-fatal — email was already dispatched.
	}

	resourceID := ""
	if ir != nil {
		resourceID = ir.ID
	}
	model.CreateAuditEntry(r.Context(), s.db, "invite_request.submitted", req.Email, "invite_request", resourceID, clientIP(r),
		map[string]any{"name": req.Name, "org": req.Org})

	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ListInviteRequestsAdmin returns stored invite requests. Admin-only.
//
// GET /api/invite/requests
// Query param: status=pending|approved|denied (omit for all)
func (s *Server) ListInviteRequestsAdmin(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	switch status {
	case "pending", "approved", "denied", "":
		// valid
	default:
		s.writeError(w, http.StatusBadRequest, "status must be pending, approved, or denied")
		return
	}

	requests, err := model.ListInviteRequests(r.Context(), s.db, status)
	if err != nil {
		log.Printf("list invite requests: %v", err)
		s.writeError(w, http.StatusInternalServerError, "failed to list invite requests")
		return
	}
	if requests == nil {
		requests = []model.InviteRequest{}
	}
	s.writeJSON(w, http.StatusOK, map[string]any{
		"requests": requests,
		"total":    len(requests),
	})
}

// ApproveInviteRequestAdmin approves a pending request from the admin dashboard,
// creating an invite code and emailing it to the requester.
//
// POST /api/invite/requests/{id}/approve
func (s *Server) ApproveInviteRequestAdmin(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ir, err := model.GetInviteRequest(r.Context(), s.db, id)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "invite request not found")
		return
	}
	if ir.Status != "pending" {
		s.writeError(w, http.StatusConflict, "invite request is not pending (status: "+ir.Status+")")
		return
	}

	ic, err := model.CreateInviteCode(r.Context(), s.db, ir.Name+" ("+ir.Email+")")
	if err != nil {
		log.Printf("invite approve admin: create code for %s: %v", ir.Email, err)
		s.writeError(w, http.StatusInternalServerError, "failed to create invite code")
		return
	}

	inviteURL := s.cfg.LandingBaseURL + "/?invite=" + ic.Code
	subject, body := email.InviteCodeIssued(ir.Name, ic.Code, inviteURL, s.cfg.LandingBaseURL)
	if sendErr := s.mailer.Send(r.Context(), ir.Email, subject, body); sendErr != nil {
		log.Printf("invite approve admin: send email to %s: %v", ir.Email, sendErr)
		// Code still created — mark approved regardless.
	}

	if markErr := model.ApproveInviteRequest(r.Context(), s.db, id, actorEmail(r), ic.ID); markErr != nil {
		log.Printf("invite approve admin: mark approved %s: %v", id, markErr)
	}

	model.CreateAuditEntry(r.Context(), s.db, "invite_code.issued", actorEmail(r), "invite_code", ic.ID, clientIP(r),
		map[string]any{"label": ic.Label, "requester": ir.Email, "request_id": id})

	s.writeJSON(w, http.StatusOK, map[string]any{
		"status":      "approved",
		"invite_code": ic,
	})
}

// DenyInviteRequestAdmin denies a pending invite request. Admin-only.
//
// POST /api/invite/requests/{id}/deny
func (s *Server) DenyInviteRequestAdmin(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ir, err := model.GetInviteRequest(r.Context(), s.db, id)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "invite request not found")
		return
	}
	if ir.Status != "pending" {
		s.writeError(w, http.StatusConflict, "invite request is not pending (status: "+ir.Status+")")
		return
	}

	if err := model.DenyInviteRequest(r.Context(), s.db, id, actorEmail(r)); err != nil {
		log.Printf("invite deny admin: mark denied %s: %v", id, err)
		s.writeError(w, http.StatusInternalServerError, "failed to deny invite request")
		return
	}

	model.CreateAuditEntry(r.Context(), s.db, "invite_request.denied", actorEmail(r), "invite_request", id, clientIP(r),
		map[string]any{"email": ir.Email})

	s.writeJSON(w, http.StatusOK, map[string]any{"status": "denied"})
}

// ApproveInviteRequest handles GET /api/invite/request/approve?t=...
// Validates the HMAC token, creates an invite code, and emails it to the requester.
// Returns a self-contained HTML confirmation page — designed to be clicked from email.
func (s *Server) ApproveInviteRequest(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimSpace(r.URL.Query().Get("t"))
	if token == "" || s.cfg.InviteRequestSecret == "" {
		s.writeApproveHTML(w, "", "", "", "Invalid or missing approval token.")
		return
	}

	p, err := verifyInviteToken(s.cfg.InviteRequestSecret, token)
	if err != nil {
		log.Printf("invite approve: token validation failed: %v", err)
		msg := "This approval link is invalid or has expired (links are valid for 7 days)."
		s.writeApproveHTML(w, "", "", "", msg)
		return
	}

	ic, err := model.CreateInviteCode(r.Context(), s.db, p.Name+" ("+p.Email+")")
	if err != nil {
		log.Printf("invite approve: create code for %s: %v", p.Email, err)
		s.writeApproveHTML(w, p.Name, p.Email, "", "Failed to create invite code. Please try again or create one manually in the admin dashboard.")
		return
	}

	inviteURL := s.cfg.LandingBaseURL + "/?invite=" + ic.Code
	subject, body := email.InviteCodeIssued(p.Name, ic.Code, inviteURL, s.cfg.LandingBaseURL)
	if err := s.mailer.Send(r.Context(), p.Email, subject, body); err != nil {
		log.Printf("invite approve: send code email to %s: %v", p.Email, err)
		// Code was created — show it on the page so admin can forward manually.
		s.writeApproveHTML(w, p.Name, p.Email, ic.Code,
			"Invite code created but the email could not be sent. Please forward the code manually.")
		return
	}

	model.CreateAuditEntry(r.Context(), s.db, "invite_code.issued", actorEmail(r), "invite_code", ic.ID, clientIP(r),
		map[string]any{"label": ic.Label, "requester": p.Email})

	s.writeApproveHTML(w, p.Name, p.Email, ic.Code, "")
}

// writeApproveHTML renders a self-contained HTML confirmation page for the approval endpoint.
// errMsg non-empty → error state; code non-empty → success with code display.
func (s *Server) writeApproveHTML(w http.ResponseWriter, name, toEmail, code, errMsg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")

	isErr := errMsg != ""
	statusIcon := "✓"
	statusColor := "#0d9488"
	statusBg := "#ccfbf1"
	statusBorder := "#0f766e"
	headingColor := "#0f766e"
	if isErr {
		statusIcon = "✗"
		statusColor = "#ea580c"
		statusBg = "#ffedd5"
		statusBorder = "#9a3412"
		headingColor = "#9a3412"
	}

	var heading, detail string
	if isErr {
		heading = "Approval failed"
		detail = errMsg
		if code != "" {
			detail += fmt.Sprintf("\n\nCode created: %s", code)
		}
	} else {
		heading = "Invite sent!"
		detail = fmt.Sprintf("An invite code has been emailed to %s (%s).\n\nCode: %s", name, toEmail, code)
	}

	// Escape for HTML embedding.
	heading = htmlEscape(heading)
	detail = htmlEscape(detail)

	fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>AEGIS — Invite Approval</title>
  <style>
    *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
    body {
      background: #0f172a; color: #e2e8f0;
      font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', system-ui, sans-serif;
      min-height: 100vh; display: flex; align-items: center; justify-content: center; padding: 24px;
    }
    .card {
      width: 100%%; max-width: 440px; background: #1e293b; border: 1px solid #334155;
      border-radius: 16px; padding: 40px 36px; box-shadow: 0 24px 48px rgba(0,0,0,.5);
    }
    .badge {
      width: 48px; height: 48px; border-radius: 50%%;
      background: %s; border: 2px solid %s; color: %s;
      font-size: 22px; display: flex; align-items: center; justify-content: center;
      margin-bottom: 20px;
    }
    h1 { font-size: 1.2rem; font-weight: 700; color: %s; margin-bottom: 12px; }
    p { font-size: 0.88rem; color: #94a3b8; line-height: 1.7; white-space: pre-wrap; }
    .code {
      margin-top: 16px; background: #0f172a; border: 1px solid #334155; border-radius: 8px;
      padding: 12px 16px; font-family: ui-monospace, 'SF Mono', monospace;
      font-size: 1.1rem; letter-spacing: 0.15em; color: #f1f5f9; text-align: center;
    }
    a { color: #3b82f6; text-decoration: none; font-size: 0.82rem; margin-top: 20px; display: inline-block; }
    a:hover { text-decoration: underline; }
  </style>
</head>
<body>
  <div class="card">
    <div class="badge">%s</div>
    <h1>%s</h1>
    <p>%s</p>
    %s
    <a href="%s">← Back to admin dashboard</a>
  </div>
</body>
</html>`,
		statusBg, statusBorder, statusColor,
		headingColor,
		statusIcon,
		heading,
		detail,
		func() string {
			if code != "" && !isErr {
				return fmt.Sprintf(`<div class="code">%s</div>`, htmlEscape(code))
			}
			return ""
		}(),
		s.cfg.AdminDashboardURL,
	)
}

// htmlEscape escapes the five HTML special characters.
func htmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&#34;")
	s = strings.ReplaceAll(s, "'", "&#39;")
	return s
}
