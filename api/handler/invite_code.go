package handler

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

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
