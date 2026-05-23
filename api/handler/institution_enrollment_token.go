package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/aegis-imaging/aegis/api/model"
)

// CreateInstitutionEnrollmentToken POST /api/institutions/{id}/enrollment-tokens
//
// Body: {"label": "...", "ttl_hours": 72 (optional)}
// Response: {"token", "id", "expires_at", "install_command"} — `token` is
// returned only here; subsequent reads only show the metadata.
func (s *Server) CreateInstitutionEnrollmentToken(w http.ResponseWriter, r *http.Request) {
	instID := r.PathValue("id")
	inst, err := model.GetInstitutionByID(r.Context(), s.db, instID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "institution not found")
		return
	}

	var req struct {
		Label    string `json:"label"`
		TTLHours int    `json:"ttl_hours"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	req.Label = strings.TrimSpace(req.Label)
	if req.TTLHours <= 0 {
		req.TTLHours = s.cfg.SpokeEnrollmentTokenTTLHours
		if req.TTLHours <= 0 {
			req.TTLHours = 72
		}
	}
	if req.TTLHours > 24*30 {
		req.TTLHours = 24 * 30 // cap at 30 days
	}

	raw, hashHex, err := model.GenerateEnrollmentToken()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "token generation failed")
		return
	}

	tok := &model.SpokeEnrollmentToken{
		InstitutionID: inst.ID,
		Label:         req.Label,
		CreatedBy:     actorEmail(r),
		ExpiresAt:     time.Now().UTC().Add(time.Duration(req.TTLHours) * time.Hour),
	}
	if err := model.CreateSpokeEnrollmentToken(r.Context(), s.db, tok, hashHex); err != nil {
		s.writeError(w, http.StatusInternalServerError, "could not create token")
		return
	}

	model.CreateAuditEntry(r.Context(), s.db, "spoke.enrollment_token_minted",
		actorEmail(r), "institution", inst.ID, clientIP(r), map[string]any{
			"token_id":   tok.ID,
			"expires_at": tok.ExpiresAt,
			"ttl_hours":  req.TTLHours,
			"label":      req.Label,
		})

	installCmd := "./bin/aegis-router-init --site-token=" + raw +
		` --site-name="` + inst.Name + `"`

	s.writeJSON(w, http.StatusCreated, map[string]any{
		"id":              tok.ID,
		"institution_id":  inst.ID,
		"institution_slug": inst.Slug,
		"label":           tok.Label,
		"expires_at":      tok.ExpiresAt,
		"token":           raw, // returned exactly once
		"install_command": installCmd,
	})
}

// ListInstitutionEnrollmentTokens GET /api/institutions/{id}/enrollment-tokens
// Returns metadata only — never the raw token.
func (s *Server) ListInstitutionEnrollmentTokens(w http.ResponseWriter, r *http.Request) {
	instID := r.PathValue("id")
	tokens, err := model.ListSpokeEnrollmentTokens(r.Context(), s.db, instID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	now := time.Now().UTC()
	type tokenRow struct {
		model.SpokeEnrollmentToken
		Status string `json:"status"`
	}
	rows := make([]tokenRow, 0, len(tokens))
	for _, t := range tokens {
		row := tokenRow{SpokeEnrollmentToken: t}
		switch {
		case t.RevokedAt != nil:
			row.Status = "revoked"
		case t.UsedAt != nil:
			row.Status = "used"
		case now.After(t.ExpiresAt):
			row.Status = "expired"
		default:
			row.Status = "active"
		}
		rows = append(rows, row)
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"tokens": rows, "count": len(rows)})
}

// RevokeInstitutionEnrollmentToken DELETE /api/institutions/{id}/enrollment-tokens/{tokenID}
func (s *Server) RevokeInstitutionEnrollmentToken(w http.ResponseWriter, r *http.Request) {
	instID := r.PathValue("id")
	tokenID := r.PathValue("tokenID")
	if err := model.RevokeSpokeEnrollmentToken(r.Context(), s.db, tokenID); err != nil {
		s.writeError(w, http.StatusInternalServerError, "revoke failed")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "spoke.enrollment_token_revoked",
		actorEmail(r), "institution", instID, clientIP(r), map[string]any{
			"token_id": tokenID,
		})
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}
