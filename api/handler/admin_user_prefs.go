package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/aegis-imaging/aegis/api/model"
)

// GetUserPreferences handles GET /api/admin-users/{id}/preferences.
// Returns preferences for the given admin user (defaults if no record exists).
func (s *Server) GetUserPreferences(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("id")

	// Verify the admin user exists.
	if _, err := model.GetAdminUserByID(r.Context(), s.db, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.writeError(w, http.StatusNotFound, "admin user not found")
			return
		}
		s.writeError(w, http.StatusInternalServerError, "failed to load user")
		return
	}

	prefs, err := model.GetAdminUserPreferences(r.Context(), s.db, userID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to load preferences")
		return
	}
	s.writeJSON(w, http.StatusOK, prefs)
}

// UpdateUserPreferences handles PUT /api/admin-users/{id}/preferences.
// Body: {"digest_frequency":"weekly","notify_events":["study.stuck","pipeline.failed"]}
func (s *Server) UpdateUserPreferences(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("id")

	// Verify the admin user exists.
	if _, err := model.GetAdminUserByID(r.Context(), s.db, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.writeError(w, http.StatusNotFound, "admin user not found")
			return
		}
		s.writeError(w, http.StatusInternalServerError, "failed to load user")
		return
	}

	var body struct {
		DigestFrequency string   `json:"digest_frequency"`
		NotifyEvents    []string `json:"notify_events"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if body.DigestFrequency == "" {
		body.DigestFrequency = "weekly"
	}
	if !model.ValidDigestFrequencies[body.DigestFrequency] {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid digest_frequency %q: must be one of none, daily, weekly, monthly", body.DigestFrequency))
		return
	}

	if body.NotifyEvents == nil {
		body.NotifyEvents = []string{}
	}
	for _, ev := range body.NotifyEvents {
		if !model.ValidNotifyEvents[ev] {
			s.writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid notify_event %q", ev))
			return
		}
	}

	prefs := &model.AdminUserPreferences{
		AdminUserID:     userID,
		DigestFrequency: body.DigestFrequency,
		NotifyEvents:    body.NotifyEvents,
	}
	if err := model.UpsertAdminUserPreferences(r.Context(), s.db, prefs); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to save preferences")
		return
	}

	model.CreateAuditEntry(r.Context(), s.db, "admin_user.preferences_updated", actorEmail(r), "admin_user", userID, clientIP(r),
		map[string]any{"digest_frequency": prefs.DigestFrequency, "notify_events": prefs.NotifyEvents})

	// Return the saved preferences.
	saved, err := model.GetAdminUserPreferences(r.Context(), s.db, userID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to reload preferences")
		return
	}
	s.writeJSON(w, http.StatusOK, saved)
}
