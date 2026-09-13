package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/routing"
)

type simulateRequest struct {
	ProjectID string `json:"project_id"` // empty = any-project rules only
	Modality  string `json:"modality"`   // e.g. "MRI"
	BodyPart  string `json:"body_part"`  // e.g. "HEAD"
	Source    string `json:"source"`     // "external" | "internal"
}

type simulatedMatchedRule struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Priority      int     `json:"priority"`
	Action        string  `json:"action"`
	DestinationID *string `json:"destination_id,omitempty"`
	// Destination name resolved for route_to rules (empty if dest not found)
	DestinationName string `json:"destination_name,omitempty"`
}

type simulateActionSummary struct {
	RequireDefacing        bool `json:"require_defacing"`
	RequirePhiScan         bool `json:"require_phi_scan"`
	RequireQcCheck         bool `json:"require_qc_check"`
	RequireBidsConversion  bool `json:"require_bids_conversion"`
	RequireClassification  bool `json:"require_classification"`
	RequireProtocolCheck   bool `json:"require_protocol_check"`
	RequireExport          bool `json:"require_export"`
	RequirePixelRedaction  bool `json:"require_pixel_redaction"`
	RequireAnalytics       bool `json:"require_analytics"`
	RequireSct             bool `json:"require_sct"`
	AutoApprove            bool `json:"auto_approve"`
	Reject                 bool `json:"reject"`
}

type simulateResponse struct {
	Input         simulateRequest        `json:"input"`
	MatchedRules  []simulatedMatchedRule `json:"matched_rules"`
	SkippedRules  []simulatedMatchedRule `json:"skipped_rules"` // enabled rules that did NOT match
	ActionSummary simulateActionSummary  `json:"action_summary"`
}

// SimulateRoutingRules performs a dry-run evaluation of all enabled routing rules
// against a hypothetical study with the supplied attributes. No data is written.
//
// POST /api/routing-rules/simulate
func (s *Server) SimulateRoutingRules(w http.ResponseWriter, r *http.Request) {
	var req simulateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	req.Modality = strings.ToUpper(strings.TrimSpace(req.Modality))
	req.BodyPart = strings.ToUpper(strings.TrimSpace(req.BodyPart))
	req.Source = strings.ToLower(strings.TrimSpace(req.Source))
	if req.Source == "" {
		req.Source = "external"
	}

	ctx := r.Context()

	// Load enabled rules.
	rules, err := model.ListEnabledRoutingRules(ctx, s.db)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to load routing rules: "+err.Error())
		return
	}

	// Build a synthetic study for matching — no ID needed, Matches() only
	// reads ProjectID, Modality, BodyPart, Source.
	study := &model.Study{
		ProjectID: req.ProjectID,
		Modality:  req.Modality,
		BodyPart:  req.BodyPart,
		Source:    req.Source,
	}

	// Pre-fetch all destinations so we can resolve names for route_to rules.
	destinations, _ := model.ListDestinations(ctx, s.db)
	destByID := make(map[string]model.Destination, len(destinations))
	for _, d := range destinations {
		destByID[d.ID] = d
	}

	var matched []simulatedMatchedRule
	var skipped []simulatedMatchedRule
	var summary simulateActionSummary

	for i := range rules {
		r := &rules[i]
		entry := simulatedMatchedRule{
			ID:            r.ID,
			Name:          r.Name,
			Priority:      r.Priority,
			Action:        r.Action,
			DestinationID: r.DestinationID,
		}
		if r.DestinationID != nil {
			if d, ok := destByID[*r.DestinationID]; ok {
				entry.DestinationName = d.Name
			}
		}

		if !routing.Matches(r, study) {
			skipped = append(skipped, entry)
			continue
		}

		matched = append(matched, entry)

		// Accumulate action summary.
		switch r.Action {
		case "require_defacing":
			summary.RequireDefacing = true
		case "require_phi_scan":
			summary.RequirePhiScan = true
		case "require_qc_check":
			summary.RequireQcCheck = true
		case "require_bids_conversion":
			summary.RequireBidsConversion = true
		case "require_classification":
			summary.RequireClassification = true
		case "require_protocol_check":
			summary.RequireProtocolCheck = true
		case "require_export":
			summary.RequireExport = true
		case "require_pixel_redaction":
			summary.RequirePixelRedaction = true
		case "require_analytics":
			summary.RequireAnalytics = true
		case "require_sct":
			summary.RequireSct = true
		case "auto_approve":
			summary.AutoApprove = true
		case "reject":
			summary.Reject = true
		}
	}

	if matched == nil {
		matched = []simulatedMatchedRule{}
	}
	if skipped == nil {
		skipped = []simulatedMatchedRule{}
	}

	s.writeJSON(w, http.StatusOK, simulateResponse{
		Input:         req,
		MatchedRules:  matched,
		SkippedRules:  skipped,
		ActionSummary: summary,
	})
}
