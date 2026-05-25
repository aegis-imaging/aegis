package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/aegis-imaging/aegis/api/model"
)

// GET /api/institutions/{id}/upload-allowlist
//
// Returns the *effective* allowlist — one entry per KnownUploadMethod
// with the resolved enabled flag. Methods with no explicit row return
// IsDefault=true and Enabled=true (per the default-on contract). UI
// renders the full list of toggles in stable order.
func (s *Server) ListInstitutionUploadAllowlist(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		s.writeError(w, http.StatusBadRequest, "institution id is required")
		return
	}
	out, err := model.ListEffectiveUploadAllowlist(r.Context(), s.db, id)
	if err != nil {
		log.Printf("list upload allowlist for %s: %v", id, err)
		s.writeError(w, http.StatusInternalServerError, "failed to list upload allowlist")
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{
		"institution_id": id,
		"entries":        out,
	})
}

type upsertUploadAllowlistRequest struct {
	Enabled *bool  `json:"enabled"`
	Note    string `json:"note"`
}

// PUT /api/institutions/{id}/upload-allowlist/{methodID}
//
// Upsert a single row. Required: `enabled` (explicit true/false, not
// inferred from absence). Optional: `note`.
func (s *Server) PutInstitutionUploadAllowlist(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	methodID := r.PathValue("methodID")
	if id == "" || methodID == "" {
		s.writeError(w, http.StatusBadRequest, "institution id and method id are required")
		return
	}
	if !model.IsKnownUploadMethod(methodID) {
		s.writeError(w, http.StatusBadRequest, "unknown upload method")
		return
	}

	var req upsertUploadAllowlistRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Enabled == nil {
		s.writeError(w, http.StatusBadRequest, "enabled is required")
		return
	}

	row := &model.InstitutionUploadAllowlistRow{
		InstitutionID: id,
		MethodID:      methodID,
		Enabled:       *req.Enabled,
		Note:          strings.TrimSpace(req.Note),
		UpdatedBy:     actorEmail(r),
	}
	if err := model.UpsertUploadAllowlistRow(r.Context(), s.db, row); err != nil {
		log.Printf("upsert upload allowlist (%s, %s): %v", id, methodID, err)
		s.writeError(w, http.StatusInternalServerError, "failed to update upload allowlist")
		return
	}

	action := "institution.upload_allowlist.enabled"
	if !*req.Enabled {
		action = "institution.upload_allowlist.disabled"
	}
	model.CreateAuditEntry(r.Context(), s.db, action, actorEmail(r), "institution", id, clientIP(r),
		map[string]any{"method_id": methodID, "note": row.Note})

	s.writeJSON(w, http.StatusOK, row)
}

// DELETE /api/institutions/{id}/upload-allowlist/{methodID}
//
// Drop the row, reverting (institution, method) to default-on. 404 if
// no row existed so the caller can tell "already at default" from
// "successfully reverted."
func (s *Server) DeleteInstitutionUploadAllowlist(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	methodID := r.PathValue("methodID")
	if id == "" || methodID == "" {
		s.writeError(w, http.StatusBadRequest, "institution id and method id are required")
		return
	}
	if !model.IsKnownUploadMethod(methodID) {
		s.writeError(w, http.StatusBadRequest, "unknown upload method")
		return
	}

	err := model.DeleteUploadAllowlistRow(r.Context(), s.db, id, methodID)
	if errors.Is(err, sql.ErrNoRows) {
		s.writeError(w, http.StatusNotFound, "no explicit allowlist row for that method")
		return
	}
	if err != nil {
		log.Printf("delete upload allowlist (%s, %s): %v", id, methodID, err)
		s.writeError(w, http.StatusInternalServerError, "failed to delete upload allowlist row")
		return
	}

	model.CreateAuditEntry(r.Context(), s.db, "institution.upload_allowlist.reset", actorEmail(r),
		"institution", id, clientIP(r), map[string]any{"method_id": methodID})

	w.WriteHeader(http.StatusNoContent)
}
