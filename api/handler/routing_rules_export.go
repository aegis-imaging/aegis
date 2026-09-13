package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/aegis-imaging/aegis/api/model"
)

// ExportRoutingRules streams all project-scoped routing rules as a JSON download.
//
// GET /api/projects/{id}/routing-rules/export
//
// Only rules where project_id = {id} are included — global rules (project_id IS NULL)
// are not exported because they belong to no single project.
func (s *Server) ExportRoutingRules(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")

	rules, err := model.ListRoutingRulesByProject(r.Context(), s.db, projectID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to list routing rules")
		return
	}
	if rules == nil {
		rules = []model.RoutingRule{}
	}

	payload := map[string]any{
		"project_id": projectID,
		"rules":      rules,
		"count":      len(rules),
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to marshal rules")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", `attachment; filename="routing-rules.json"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
	model.CreateAuditEntry(r.Context(), s.db, "routing_rule.exported", actorEmail(r), "project", projectID, clientIP(r),
		map[string]any{"count": len(rules)})
}

// ImportRoutingRules bulk-creates routing rules from a JSON payload.
//
// POST /api/projects/{id}/routing-rules/import
//
// Accepts either a raw JSON array of rule objects or the export format
// {"rules":[...]}. Rules whose name already exists in the project are
// skipped (not overwritten).
//
// Because destination IDs are instance-specific, any destination_id value
// from the import payload is stripped — operators must re-assign route_to
// destinations after import via the Routing tab.
func (s *Server) ImportRoutingRules(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	ctx := r.Context()

	if _, err := model.GetProjectByID(ctx, s.db, projectID); err != nil {
		s.writeError(w, http.StatusNotFound, "project not found")
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 2<<20)) // 2 MB limit
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "read body: "+err.Error())
		return
	}

	// Accept both the export format {"rules":[...]} and a raw JSON array.
	var incoming []model.RoutingRule
	var exportPayload struct {
		Rules []model.RoutingRule `json:"rules"`
	}
	if err := json.Unmarshal(body, &exportPayload); err == nil && exportPayload.Rules != nil {
		incoming = exportPayload.Rules
	} else if err := json.Unmarshal(body, &incoming); err != nil {
		s.writeError(w, http.StatusBadRequest, `invalid JSON: expected array or {"rules":[...]}`)
		return
	}

	if len(incoming) == 0 {
		s.writeJSON(w, http.StatusOK, map[string]any{"imported": 0, "skipped": 0, "errors": []string{}})
		return
	}

	// Build a set of existing names to detect duplicates.
	existing, err := model.ListRoutingRulesByProject(ctx, s.db, projectID)
	if err != nil {
		log.Printf("import routing rules: list existing: %v", err)
		s.writeError(w, http.StatusInternalServerError, "failed to check existing rules")
		return
	}
	seen := make(map[string]bool, len(existing))
	for _, rule := range existing {
		seen[strings.ToLower(rule.Name)] = true
	}

	var imported, skipped int
	var errs []string

	for i, src := range incoming {
		name := strings.TrimSpace(src.Name)
		if name == "" {
			errs = append(errs, fmt.Sprintf("rule[%d]: name is required", i))
			continue
		}
		if src.Action == "" {
			errs = append(errs, fmt.Sprintf("rule[%d] %q: action is required", i, name))
			continue
		}
		if seen[strings.ToLower(name)] {
			skipped++
			continue
		}

		newRule := model.RoutingRule{
			Name:        name,
			Description: src.Description,
			Priority:    src.Priority,
			Enabled:     src.Enabled,
			ProjectID:   &projectID,
			Modality:    src.Modality,
			BodyPart:    src.BodyPart,
			Source:      src.Source,
			Action:      src.Action,
			// DestinationID intentionally omitted — destination IDs are
			// instance-specific and cannot be transferred across deployments.
		}

		if err := model.CreateRoutingRule(ctx, s.db, &newRule); err != nil {
			log.Printf("import routing rules: create rule %q: %v", name, err)
			errs = append(errs, fmt.Sprintf("rule[%d] %q: %v", i, name, err))
			continue
		}
		imported++
		seen[strings.ToLower(name)] = true // prevent duplicate within same import batch
	}

	model.CreateAuditEntry(ctx, s.db, "routing_rule.imported", actorEmail(r), "project", projectID, clientIP(r),
		map[string]any{"imported": imported, "skipped": skipped, "errors": len(errs)})

	s.writeJSON(w, http.StatusOK, map[string]any{
		"imported": imported,
		"skipped":  skipped,
		"errors":   errs,
	})
}
