package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/aegis-imaging/aegis/api/model"
)

// Multi-tenant SaaS scaffolding (chunk 1).
//
// These handlers are intentionally narrow: just enough CRUD to register
// tenants and look them up. Tenant-scoping of every other resource (the
// hard part) is deferred to subsequent PRs that will incrementally add
// tenant_id columns + middleware-driven filtering.

type tenantRequest struct {
	Slug     string          `json:"slug"`
	Name     string          `json:"name"`
	Settings json.RawMessage `json:"settings,omitempty"`
	Enabled  *bool           `json:"enabled,omitempty"`
}

// ListTenants GET /api/tenants
func (s *Server) ListTenants(w http.ResponseWriter, r *http.Request) {
	tenants, err := model.ListTenants(r.Context(), s.db)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	if tenants == nil {
		tenants = []model.Tenant{}
	}
	s.writeJSON(w, http.StatusOK, tenants)
}

// CreateTenant POST /api/tenants
func (s *Server) CreateTenant(w http.ResponseWriter, r *http.Request) {
	var req tenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	t := &model.Tenant{
		Slug:     req.Slug,
		Name:     req.Name,
		Settings: req.Settings,
	}
	// model.CreateTenant always creates an enabled tenant (uses the DB column
	// default) — see the doc comment there. If the API caller explicitly
	// asked for `enabled: false`, we honor it via a follow-up update.
	if err := model.CreateTenant(r.Context(), s.db, t); err != nil {
		// Slug-validation / name-required errors come back as plain errors;
		// surface them to the client so they know what to fix.
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Enabled != nil && !*req.Enabled {
		t.Enabled = false
		if err := model.UpdateTenant(r.Context(), s.db, t); err != nil {
			s.writeError(w, http.StatusInternalServerError, "failed to apply enabled=false")
			return
		}
	}
	model.CreateAuditEntry(r.Context(), s.db, "tenant.created", actorEmail(r), "tenant", t.ID, clientIP(r), map[string]any{
		"slug": t.Slug, "name": t.Name,
	})
	s.writeJSON(w, http.StatusCreated, t)
}

// GetTenant GET /api/tenants/{id}
func (s *Server) GetTenant(w http.ResponseWriter, r *http.Request) {
	t, err := model.GetTenant(r.Context(), s.db, r.PathValue("id"))
	if err != nil {
		if errors.Is(err, model.ErrTenantNotFound) {
			s.writeError(w, http.StatusNotFound, "tenant not found")
			return
		}
		s.writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	s.writeJSON(w, http.StatusOK, t)
}

// UpdateTenant PUT /api/tenants/{id}
func (s *Server) UpdateTenant(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	t, err := model.GetTenant(r.Context(), s.db, id)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "tenant not found")
		return
	}
	var req tenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	// Slug is intentionally immutable post-create — see model.UpdateTenant.
	if req.Name != "" {
		t.Name = req.Name
	}
	if len(req.Settings) > 0 {
		t.Settings = req.Settings
	}
	if req.Enabled != nil {
		t.Enabled = *req.Enabled
	}
	if err := model.UpdateTenant(r.Context(), s.db, t); err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "tenant.updated", actorEmail(r), "tenant", id, clientIP(r), nil)
	s.writeJSON(w, http.StatusOK, t)
}

// DeleteTenant DELETE /api/tenants/{id}
func (s *Server) DeleteTenant(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := model.GetTenant(r.Context(), s.db, id); err != nil {
		s.writeError(w, http.StatusNotFound, "tenant not found")
		return
	}
	if err := model.DeleteTenant(r.Context(), s.db, id); err != nil {
		// Postgres returns a foreign-key error when projects still reference
		// the tenant. Surface it as a 409 so the caller knows to re-home
		// projects first.
		s.writeError(w, http.StatusConflict, "tenant still has projects — re-home or delete them first")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "tenant.deleted", actorEmail(r), "tenant", id, clientIP(r), nil)
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
