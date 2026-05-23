package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/aegis-imaging/aegis/api/model"
)

// ListDestinationMaintenance GET /api/destinations/{id}/maintenance
func (s *Server) ListDestinationMaintenance(w http.ResponseWriter, r *http.Request) {
	destID := r.PathValue("id")
	windows, err := model.ListDestinationMaintenance(r.Context(), s.db, destID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	if windows == nil {
		windows = []model.DestinationMaintenanceWindow{}
	}
	s.writeJSON(w, http.StatusOK, windows)
}

// CreateDestinationMaintenance POST /api/destinations/{id}/maintenance
func (s *Server) CreateDestinationMaintenance(w http.ResponseWriter, r *http.Request) {
	destID := r.PathValue("id")

	var req struct {
		Reason   string `json:"reason"`
		StartsAt string `json:"starts_at"`
		EndsAt   string `json:"ends_at"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	startsAt, err := time.Parse(time.RFC3339, strings.TrimSpace(req.StartsAt))
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "starts_at must be RFC3339 format")
		return
	}
	endsAt, err := time.Parse(time.RFC3339, strings.TrimSpace(req.EndsAt))
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "ends_at must be RFC3339 format")
		return
	}
	if !endsAt.After(startsAt) {
		s.writeError(w, http.StatusBadRequest, "ends_at must be after starts_at")
		return
	}

	m := &model.DestinationMaintenanceWindow{
		DestinationID: destID,
		Reason:        strings.TrimSpace(req.Reason),
		StartsAt:      startsAt,
		EndsAt:        endsAt,
		CreatedBy:     actorEmail(r),
	}
	if err := model.CreateDestinationMaintenance(r.Context(), s.db, m); err != nil {
		s.writeError(w, http.StatusInternalServerError, "create failed")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "destination.maintenance_scheduled", actorEmail(r), "destination", destID, clientIP(r), map[string]any{
		"starts_at": req.StartsAt, "ends_at": req.EndsAt,
	})
	s.writeJSON(w, http.StatusCreated, m)
}

// DeleteDestinationMaintenance DELETE /api/destinations/{id}/maintenance/{windowID}
func (s *Server) DeleteDestinationMaintenance(w http.ResponseWriter, r *http.Request) {
	destID := r.PathValue("id")
	windowID := r.PathValue("windowID")
	if err := model.DeleteDestinationMaintenance(r.Context(), s.db, windowID, destID); err != nil {
		s.writeError(w, http.StatusInternalServerError, "delete failed")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "destination.maintenance_cancelled", actorEmail(r), "destination", destID, clientIP(r), map[string]any{
		"window_id": windowID,
	})
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
