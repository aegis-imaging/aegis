package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/routing"
)

// ─── Destinations ────────────────────────────────────────────────────────────

func (s *Server) ListDestinations(w http.ResponseWriter, r *http.Request) {
	dests, err := model.ListDestinations(r.Context(), s.db)
	if err != nil {
		log.Printf("list destinations: %v", err)
		s.writeError(w, http.StatusInternalServerError, "failed to list destinations")
		return
	}
	if dests == nil {
		dests = []model.Destination{}
	}
	s.writeJSON(w, http.StatusOK, dests)
}

func (s *Server) CreateDestination(w http.ResponseWriter, r *http.Request) {
	var d model.Destination
	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if d.Name == "" || d.Type == "" {
		s.writeError(w, http.StatusBadRequest, "name and type are required")
		return
	}
	if d.Type != "dicomweb" && d.Type != "dimse" {
		s.writeError(w, http.StatusBadRequest, "type must be dicomweb or dimse")
		return
	}
	if err := validateDestination(&d); err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if d.Slug == "" {
		d.Slug = slugify(d.Name)
	}
	d.Enabled = true

	if err := model.CreateDestination(r.Context(), s.db, &d); err != nil {
		log.Printf("create destination: %v", err)
		s.writeError(w, http.StatusInternalServerError, "failed to create destination")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "destination.created", actorEmail(r), "destination", d.ID, clientIP(r), map[string]any{
		"name": d.Name,
		"type": d.Type,
	})
	s.writeJSON(w, http.StatusCreated, d)
}

func (s *Server) UpdateDestination(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	existing, err := model.GetDestinationByID(r.Context(), s.db, id)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "destination not found")
		return
	}

	var update model.Destination
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	existing.Name = update.Name
	existing.Slug = update.Slug
	existing.Description = update.Description
	existing.Type = update.Type
	existing.DicomwebURL = update.DicomwebURL
	existing.DicomwebAuthHeader = update.DicomwebAuthHeader
	existing.AETitle = update.AETitle
	existing.Host = update.Host
	existing.Port = update.Port
	existing.Enabled = update.Enabled
	if err := validateDestination(existing); err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := model.UpdateDestination(r.Context(), s.db, existing); err != nil {
		log.Printf("update destination %s: %v", id, err)
		s.writeError(w, http.StatusInternalServerError, "failed to update destination")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "destination.updated", actorEmail(r), "destination", id, clientIP(r), nil)
	s.writeJSON(w, http.StatusOK, existing)
}

func (s *Server) DeleteDestination(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := model.GetDestinationByID(r.Context(), s.db, id); err != nil {
		s.writeError(w, http.StatusNotFound, "destination not found")
		return
	}
	if err := model.DeleteDestination(r.Context(), s.db, id); err != nil {
		log.Printf("delete destination %s: %v", id, err)
		s.writeError(w, http.StatusInternalServerError, "failed to delete destination")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "destination.deleted", actorEmail(r), "destination", id, clientIP(r), nil)
	w.WriteHeader(http.StatusNoContent)
}

// ─── Routing Rules ────────────────────────────────────────────────────────────

func (s *Server) ListRoutingRules(w http.ResponseWriter, r *http.Request) {
	rules, err := model.ListRoutingRules(r.Context(), s.db)
	if err != nil {
		log.Printf("list routing rules: %v", err)
		s.writeError(w, http.StatusInternalServerError, "failed to list routing rules")
		return
	}
	if rules == nil {
		rules = []model.RoutingRule{}
	}
	s.writeJSON(w, http.StatusOK, rules)
}

func (s *Server) CreateRoutingRule(w http.ResponseWriter, r *http.Request) {
	var rule model.RoutingRule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if rule.Name == "" {
		s.writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	validActions := map[string]bool{
		"route_to": true, "require_defacing": true,
		"auto_approve": true, "require_qa": true, "reject": true,
		"require_phi_scan": true, "require_qc_check": true,
		"require_bids_conversion": true, "require_classification": true,
		"require_protocol_check": true, "require_pixel_redaction": true,
		"require_analytics": true,
	}
	if !validActions[rule.Action] {
		s.writeError(w, http.StatusBadRequest, "invalid action")
		return
	}
	if rule.Action == "route_to" && rule.DestinationID == nil {
		s.writeError(w, http.StatusBadRequest, "destination_id required for route_to action")
		return
	}
	if rule.Priority == 0 {
		rule.Priority = 100
	}
	rule.Enabled = true

	if err := model.CreateRoutingRule(r.Context(), s.db, &rule); err != nil {
		log.Printf("create routing rule: %v", err)
		s.writeError(w, http.StatusInternalServerError, "failed to create routing rule")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "routing_rule.created", actorEmail(r), "routing_rule", rule.ID, clientIP(r), map[string]any{
		"name":   rule.Name,
		"action": rule.Action,
	})
	s.writeJSON(w, http.StatusCreated, rule)
}

func (s *Server) UpdateRoutingRule(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	existing, err := model.GetRoutingRuleByID(r.Context(), s.db, id)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "routing rule not found")
		return
	}

	var update model.RoutingRule
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	existing.Name = update.Name
	existing.Description = update.Description
	existing.Priority = update.Priority
	existing.Enabled = update.Enabled
	existing.ProjectID = update.ProjectID
	existing.Modality = update.Modality
	existing.BodyPart = update.BodyPart
	existing.Source = update.Source
	existing.Action = update.Action
	existing.DestinationID = update.DestinationID

	if err := model.UpdateRoutingRule(r.Context(), s.db, existing); err != nil {
		log.Printf("update routing rule %s: %v", id, err)
		s.writeError(w, http.StatusInternalServerError, "failed to update routing rule")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "routing_rule.updated", actorEmail(r), "routing_rule", id, clientIP(r), nil)
	s.writeJSON(w, http.StatusOK, existing)
}

func (s *Server) DeleteRoutingRule(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := model.GetRoutingRuleByID(r.Context(), s.db, id); err != nil {
		s.writeError(w, http.StatusNotFound, "routing rule not found")
		return
	}
	if err := model.DeleteRoutingRule(r.Context(), s.db, id); err != nil {
		log.Printf("delete routing rule %s: %v", id, err)
		s.writeError(w, http.StatusInternalServerError, "failed to delete routing rule")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "routing_rule.deleted", actorEmail(r), "routing_rule", id, clientIP(r), nil)
	w.WriteHeader(http.StatusNoContent)
}

// EvaluateRoutingRules manually triggers rule evaluation for a study.
// Useful for re-evaluating after rules change, or for testing.
func (s *Server) EvaluateRoutingRules(w http.ResponseWriter, r *http.Request) {
	studyID := r.PathValue("studyID")
	study, err := model.GetStudyByID(r.Context(), s.db, studyID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "study not found")
		return
	}

	routing.EvaluateRules(r.Context(), s.db, s.store, study)

	log, err := model.ListRoutingLogForStudy(r.Context(), s.db, study.ID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to read routing log")
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{
		"study":       study,
		"routing_log": log,
	})
}

// GetStudyRoutingLog returns the routing log for a specific study.
func (s *Server) GetStudyRoutingLog(w http.ResponseWriter, r *http.Request) {
	studyID := r.PathValue("studyID")
	if _, err := model.GetStudyByID(r.Context(), s.db, studyID); err != nil {
		s.writeError(w, http.StatusNotFound, "study not found")
		return
	}
	entries, err := model.ListRoutingLogForStudy(r.Context(), s.db, studyID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to read routing log")
		return
	}
	if entries == nil {
		entries = []model.RoutingLogEntry{}
	}
	s.writeJSON(w, http.StatusOK, entries)
}

// slugify converts a human-readable name to a URL-safe slug.
func slugify(name string) string {
	lower := strings.ToLower(name)
	result := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		if r == ' ' || r == '_' {
			return '-'
		}
		return -1
	}, lower)
	return strings.Trim(result, "-")
}

func validateDestination(d *model.Destination) error {
	if d.Type == "dicomweb" {
		if strings.TrimSpace(d.DicomwebURL) == "" {
			return fmt.Errorf("dicomweb_url is required for dicomweb destinations")
		}
		return nil
	}
	if d.Type == "dimse" {
		if strings.TrimSpace(d.AETitle) == "" {
			return fmt.Errorf("ae_title is required for dimse destinations")
		}
		if strings.TrimSpace(d.Host) == "" {
			return fmt.Errorf("host is required for dimse destinations")
		}
		if d.Port <= 0 {
			return fmt.Errorf("port must be > 0 for dimse destinations")
		}
		return nil
	}
	return fmt.Errorf("type must be dicomweb or dimse")
}

type destinationTestResult struct {
	DestinationID string   `json:"destination_id"`
	Type          string   `json:"type"`
	Success       bool     `json:"success"`
	LatencyMs     float64  `json:"latency_ms"`
	StatusCode    *int     `json:"status_code,omitempty"`
	Error         string   `json:"error,omitempty"`
}

// TestDestination probes an external DICOM destination to verify connectivity.
// DICOMweb: sends GET {dicomweb_url}/studies?limit=1; accepts any <400 or 405
// (STOW-RS endpoints only accept POST so 405 = reachable, not an error).
// DIMSE: sends C-ECHO via the dimse-receiver /echo endpoint.
func (s *Server) TestDestination(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	dest, err := model.GetDestinationByID(r.Context(), s.db, id)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "destination not found")
		return
	}

	res := s.probeDestination(r.Context(), *dest)

	result := destinationTestResult{
		DestinationID: dest.ID,
		Type:          dest.Type,
		Success:       res.Success,
		LatencyMs:     res.LatencyMs,
		StatusCode:    res.StatusCode,
		Error:         res.Error,
	}

	meta := map[string]any{"type": dest.Type, "success": res.Success, "latency_ms": res.LatencyMs}
	if res.Error != "" {
		meta["error"] = res.Error
	}
	model.CreateAuditEntry(r.Context(), s.db, "destination.tested", actorEmail(r), "destination", id, clientIP(r), meta)
	s.writeJSON(w, http.StatusOK, result)
}
