package handler

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"github.com/aegis-imaging/aegis/api/model"
)

var federationSlugRe = regexp.MustCompile(`[^a-z0-9]+`)

type federationPeerRequest struct {
	Name   string `json:"name"`
	Slug   string `json:"slug"`
	APIURL string `json:"api_url"`
	Notes  string `json:"notes"`
}

// ListFederationPeers returns all configured federation peers.
// GET /api/federation-peers
func (s *Server) ListFederationPeers(w http.ResponseWriter, r *http.Request) {
	peers, err := model.ListFederationPeers(r.Context(), s.db)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to list federation peers")
		return
	}
	if peers == nil {
		peers = []model.FederationPeer{}
	}
	s.writeJSON(w, http.StatusOK, peers)
}

// GetFederationPeer returns a single federation peer by ID.
// GET /api/federation-peers/{id}
func (s *Server) GetFederationPeer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	peer, err := model.GetFederationPeer(r.Context(), s.db, id)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "federation peer not found")
		return
	}
	s.writeJSON(w, http.StatusOK, peer)
}

// CreateFederationPeer creates a new federation peer.
// POST /api/federation-peers
func (s *Server) CreateFederationPeer(w http.ResponseWriter, r *http.Request) {
	var req federationPeerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Name == "" {
		s.writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.APIURL == "" {
		s.writeError(w, http.StatusBadRequest, "api_url is required")
		return
	}
	if req.Slug == "" {
		req.Slug = federationSlugRe.ReplaceAllString(strings.ToLower(req.Name), "-")
		req.Slug = strings.Trim(req.Slug, "-")
	}
	peer, err := model.CreateFederationPeer(r.Context(), s.db, req.Name, req.Slug, req.APIURL, req.Notes)
	if err != nil {
		s.writeError(w, http.StatusConflict, "federation peer slug already exists or create failed")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "federation_peer.created", actorEmail(r),
		"federation_peer", peer.ID, clientIP(r), map[string]any{"name": peer.Name, "api_url": peer.APIURL})
	s.writeJSON(w, http.StatusCreated, peer)
}

// UpdateFederationPeer updates an existing federation peer.
// PUT /api/federation-peers/{id}
func (s *Server) UpdateFederationPeer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := model.GetFederationPeer(r.Context(), s.db, id); err != nil {
		s.writeError(w, http.StatusNotFound, "federation peer not found")
		return
	}
	var req struct {
		federationPeerRequest
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Name == "" {
		s.writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.APIURL == "" {
		s.writeError(w, http.StatusBadRequest, "api_url is required")
		return
	}
	if req.Slug == "" {
		req.Slug = federationSlugRe.ReplaceAllString(strings.ToLower(req.Name), "-")
		req.Slug = strings.Trim(req.Slug, "-")
	}
	peer, err := model.UpdateFederationPeer(r.Context(), s.db, id, req.Name, req.Slug, req.APIURL, req.Notes, req.Enabled)
	if err != nil {
		s.writeError(w, http.StatusConflict, "slug already exists or update failed")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "federation_peer.updated", actorEmail(r),
		"federation_peer", id, clientIP(r), map[string]any{"name": peer.Name, "enabled": peer.Enabled})
	s.writeJSON(w, http.StatusOK, peer)
}

// DeleteFederationPeer removes a federation peer.
// DELETE /api/federation-peers/{id}
func (s *Server) DeleteFederationPeer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := model.GetFederationPeer(r.Context(), s.db, id); err != nil {
		s.writeError(w, http.StatusNotFound, "federation peer not found")
		return
	}
	if err := model.DeleteFederationPeer(r.Context(), s.db, id); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to delete federation peer")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "federation_peer.deleted", actorEmail(r),
		"federation_peer", id, clientIP(r), nil)
	w.WriteHeader(http.StatusNoContent)
}
