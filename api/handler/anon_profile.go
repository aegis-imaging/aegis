package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/aegis-imaging/aegis/api/model"
)

// ListAnonProfiles returns all anonymization profiles for a project.
// GET /api/projects/{projectID}/anon-profiles
func (s *Server) ListAnonProfiles(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("projectID")
	profiles, err := model.ListAnonProfilesByProject(r.Context(), s.db, projectID)
	if err != nil {
		log.Printf("list anon profiles: %v", err)
		s.writeError(w, http.StatusInternalServerError, "failed to list profiles")
		return
	}
	if profiles == nil {
		profiles = []model.AnonProfile{}
	}
	s.writeJSON(w, http.StatusOK, profiles)
}

// CreateAnonProfile creates a new anonymization profile for a project.
// POST /api/projects/{projectID}/anon-profiles
func (s *Server) CreateAnonProfile(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("projectID")
	if _, err := model.GetProjectByID(r.Context(), s.db, projectID); err != nil {
		s.writeError(w, http.StatusNotFound, "project not found")
		return
	}

	var p model.AnonProfile
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if p.Name == "" {
		s.writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	p.ProjectID = projectID
	p.Enabled = true

	if err := model.CreateAnonProfile(r.Context(), s.db, &p); err != nil {
		log.Printf("create anon profile: %v", err)
		s.writeError(w, http.StatusInternalServerError, "failed to create profile")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "anon_profile.created", actorEmail(r), "anon_profile", p.ID, clientIP(r), map[string]any{
		"name":       p.Name,
		"project_id": projectID,
	})
	s.writeJSON(w, http.StatusCreated, p)
}

// GetAnonProfile returns a single anonymization profile by ID.
// GET /api/anon-profiles/{id}
func (s *Server) GetAnonProfile(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	p, err := model.GetAnonProfileByID(r.Context(), s.db, id)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "profile not found")
		return
	}
	s.writeJSON(w, http.StatusOK, p)
}

// UpdateAnonProfile updates an existing anonymization profile.
// PUT /api/anon-profiles/{id}
func (s *Server) UpdateAnonProfile(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	existing, err := model.GetAnonProfileByID(r.Context(), s.db, id)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "profile not found")
		return
	}

	var update model.AnonProfile
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	existing.Name = update.Name
	existing.Description = update.Description
	existing.RetainedTags = update.RetainedTags
	existing.KeepPrivateTags = update.KeepPrivateTags
	existing.Enabled = update.Enabled

	if err := model.UpdateAnonProfile(r.Context(), s.db, existing); err != nil {
		log.Printf("update anon profile %s: %v", id, err)
		s.writeError(w, http.StatusInternalServerError, "failed to update profile")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "anon_profile.updated", actorEmail(r), "anon_profile", id, clientIP(r), nil)
	s.writeJSON(w, http.StatusOK, existing)
}

// DeleteAnonProfile deletes an anonymization profile.
// DELETE /api/anon-profiles/{id}
func (s *Server) DeleteAnonProfile(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := model.GetAnonProfileByID(r.Context(), s.db, id); err != nil {
		s.writeError(w, http.StatusNotFound, "profile not found")
		return
	}
	if err := model.DeleteAnonProfile(r.Context(), s.db, id); err != nil {
		log.Printf("delete anon profile %s: %v", id, err)
		s.writeError(w, http.StatusInternalServerError, "failed to delete profile")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "anon_profile.deleted", actorEmail(r), "anon_profile", id, clientIP(r), nil)
	w.WriteHeader(http.StatusNoContent)
}

// SetDefaultAnonProfile sets the default anonymization profile for a project.
// PUT /api/projects/{projectID}/default-anon-profile
// Body: {"profile_id": "<uuid>"} or {"profile_id": ""} to clear.
func (s *Server) SetDefaultAnonProfile(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("projectID")
	if _, err := model.GetProjectByID(r.Context(), s.db, projectID); err != nil {
		s.writeError(w, http.StatusNotFound, "project not found")
		return
	}

	var body struct {
		ProfileID string `json:"profile_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	// If a profile ID was given, verify it belongs to this project.
	if body.ProfileID != "" {
		p, err := model.GetAnonProfileByID(r.Context(), s.db, body.ProfileID)
		if err != nil || p.ProjectID != projectID {
			s.writeError(w, http.StatusBadRequest, "profile not found in this project")
			return
		}
	}

	if err := model.SetProjectDefaultAnonProfile(r.Context(), s.db, projectID, body.ProfileID); err != nil {
		log.Printf("set default anon profile: %v", err)
		s.writeError(w, http.StatusInternalServerError, "failed to update project")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "project.default_anon_profile_set", actorEmail(r), "project", projectID, clientIP(r), map[string]any{
		"profile_id": body.ProfileID,
	})
	w.WriteHeader(http.StatusNoContent)
}

// GetDefaultAnonProfile returns the active default anonymization profile for a project
// identified by slug. Returns 200 with the profile, or 204 if none is configured.
// GET /api/projects/{slug}/active-anon-profile
func (s *Server) GetDefaultAnonProfile(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	profile, err := model.GetProjectDefaultAnonProfile(r.Context(), s.db, slug)
	if err != nil {
		log.Printf("get default anon profile for %s: %v", slug, err)
		s.writeError(w, http.StatusInternalServerError, "failed to load profile")
		return
	}
	if profile == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	s.writeJSON(w, http.StatusOK, profile)
}
