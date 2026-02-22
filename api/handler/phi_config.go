package handler

import (
	"encoding/json"
	"net/http"

	"github.com/aegis-imaging/aegis/api/model"
)

// GetProjectPhiConfig GET /api/projects/{id}/phi-config
func (s *Server) GetProjectPhiConfig(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	cfg, err := model.GetProjectPhiConfig(r.Context(), s.db, projectID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	s.writeJSON(w, http.StatusOK, cfg)
}

type updatePhiConfigRequest struct {
	ConfidenceThreshold *float64 `json:"confidence_threshold"`
	MinTextLength       *int     `json:"min_text_length"`
}

// UpdateProjectPhiConfig PUT /api/projects/{id}/phi-config
func (s *Server) UpdateProjectPhiConfig(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")

	var req updatePhiConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Load current values as defaults.
	cur, err := model.GetProjectPhiConfig(r.Context(), s.db, projectID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to load config")
		return
	}

	threshold := cur.ConfidenceThreshold
	minLen := cur.MinTextLength

	if req.ConfidenceThreshold != nil {
		if *req.ConfidenceThreshold < 0 || *req.ConfidenceThreshold > 1 {
			s.writeError(w, http.StatusBadRequest, "confidence_threshold must be between 0 and 1")
			return
		}
		threshold = *req.ConfidenceThreshold
	}
	if req.MinTextLength != nil {
		if *req.MinTextLength < 1 {
			s.writeError(w, http.StatusBadRequest, "min_text_length must be at least 1")
			return
		}
		minLen = *req.MinTextLength
	}

	cfg, err := model.UpsertProjectPhiConfig(r.Context(), s.db, projectID, threshold, minLen)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "update failed")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "phi_config.updated", actorEmail(r), "project", projectID, clientIP(r), map[string]any{
		"confidence_threshold": threshold,
		"min_text_length":      minLen,
	})
	s.writeJSON(w, http.StatusOK, cfg)
}
