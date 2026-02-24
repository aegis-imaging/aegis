package handler

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/aegis-imaging/aegis/api/model"
)

type createAPIKeyRequest struct {
	Name      string  `json:"name"`
	ExpiresAt *string `json:"expires_at"` // optional RFC3339
}

type createAPIKeyResponse struct {
	*model.APIKey
	RawKey string `json:"key"` // shown once, never stored
}

// ListAPIKeys GET /api/api-keys
func (s *Server) ListAPIKeys(w http.ResponseWriter, r *http.Request) {
	keys, err := model.ListAPIKeys(r.Context(), s.db)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	if keys == nil {
		keys = []model.APIKey{}
	}
	s.writeJSON(w, http.StatusOK, keys)
}

// CreateAPIKey POST /api/api-keys
func (s *Server) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	var req createAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		s.writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	var expiresAt *time.Time
	if req.ExpiresAt != nil && *req.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err != nil {
			s.writeError(w, http.StatusBadRequest, "expires_at must be RFC3339 (e.g. 2027-01-01T00:00:00Z)")
			return
		}
		t = t.UTC()
		if !t.After(time.Now().UTC()) {
			s.writeError(w, http.StatusBadRequest, "expires_at must be in the future")
			return
		}
		expiresAt = &t
	}

	// Generate a cryptographically random 32-byte key, encode as base64url.
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to generate key")
		return
	}
	rawKey := "aegis_" + base64.RawURLEncoding.EncodeToString(buf)
	h := sha256.Sum256([]byte(rawKey))
	keyHash := hex.EncodeToString(h[:])
	keyPrefix := rawKey[:min(14, len(rawKey))]

	k, err := model.CreateAPIKey(r.Context(), s.db, name, keyHash, keyPrefix, actorEmail(r), expiresAt)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "create failed")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "api_key.created", actorEmail(r), "api_key", k.ID, clientIP(r), map[string]any{
		"name":   name,
		"prefix": keyPrefix,
	})
	s.writeJSON(w, http.StatusCreated, createAPIKeyResponse{APIKey: k, RawKey: rawKey})
}

// EnableAPIKey PATCH /api/api-keys/{id}/enable
func (s *Server) EnableAPIKey(w http.ResponseWriter, r *http.Request) {
	s.setAPIKeyEnabled(w, r, true)
}

// DisableAPIKey PATCH /api/api-keys/{id}/disable
func (s *Server) DisableAPIKey(w http.ResponseWriter, r *http.Request) {
	s.setAPIKeyEnabled(w, r, false)
}

func (s *Server) setAPIKeyEnabled(w http.ResponseWriter, r *http.Request, enabled bool) {
	id := r.PathValue("id")
	if err := model.UpdateAPIKeyEnabled(r.Context(), s.db, id, enabled); err != nil {
		s.writeError(w, http.StatusInternalServerError, "update failed")
		return
	}
	action := "api_key.enabled"
	if !enabled {
		action = "api_key.disabled"
	}
	model.CreateAuditEntry(r.Context(), s.db, action, actorEmail(r), "api_key", id, clientIP(r), nil)
	s.writeJSON(w, http.StatusOK, map[string]any{"id": id, "enabled": enabled})
}

// RotateAPIKey POST /api/api-keys/{id}/rotate
// Generates a new random key value, updates the stored hash and prefix, and
// returns the new raw key exactly once. Useful for periodic key refresh without
// a delete+create gap that would interrupt dependent services.
func (s *Server) RotateAPIKey(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	// Generate a new cryptographically random key.
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to generate key")
		return
	}
	rawKey := "aegis_" + base64.RawURLEncoding.EncodeToString(buf)
	h := sha256.Sum256([]byte(rawKey))
	keyHash := hex.EncodeToString(h[:])
	keyPrefix := rawKey[:min(14, len(rawKey))]

	k, err := model.RotateAPIKey(r.Context(), s.db, id, keyHash, keyPrefix)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "key not found")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "api_key.rotated", actorEmail(r), "api_key", id, clientIP(r), map[string]any{
		"name":   k.Name,
		"prefix": keyPrefix,
	})
	s.writeJSON(w, http.StatusOK, createAPIKeyResponse{APIKey: k, RawKey: rawKey})
}

// DeleteAPIKey DELETE /api/api-keys/{id}
func (s *Server) DeleteAPIKey(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := model.DeleteAPIKey(r.Context(), s.db, id); err != nil {
		s.writeError(w, http.StatusInternalServerError, "delete failed")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "api_key.deleted", actorEmail(r), "api_key", id, clientIP(r), nil)
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// min returns the smaller of a and b (Go <1.21 compat helper).
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
