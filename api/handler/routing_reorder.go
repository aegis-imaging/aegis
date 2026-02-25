package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/aegis-imaging/aegis/api/model"
)

// ReorderRoutingRules updates the priority of multiple routing rules in one
// atomic operation. Each entry specifies a rule ID and its new priority.
//
// POST /api/routing-rules/reorder
//
// Request body:
//
//	{"rules": [{"id": "<uuid>", "priority": 10}, {"id": "<uuid>", "priority": 20}, ...]}
//
// The caller is responsible for providing a consistent, non-conflicting set of
// priorities. The endpoint performs all updates in a single database transaction
// to avoid transient ordering inconsistencies. Returns the updated rules ordered
// by new priority.
func (s *Server) ReorderRoutingRules(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Rules []struct {
			ID       string `json:"id"`
			Priority int    `json:"priority"`
		} `json:"rules"`
	}

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if len(body.Rules) == 0 {
		s.writeError(w, http.StatusBadRequest, "rules must be a non-empty array")
		return
	}
	if len(body.Rules) > 200 {
		s.writeError(w, http.StatusBadRequest, "too many rules: max 200 per call")
		return
	}

	// Validate: all IDs must be non-empty; priorities must be ≥ 0.
	seen := make(map[string]bool, len(body.Rules))
	for i, entry := range body.Rules {
		if entry.ID == "" {
			s.writeError(w, http.StatusBadRequest, "rule entry missing id")
			return
		}
		if entry.Priority < 0 {
			s.writeError(w, http.StatusBadRequest, "priority must be ≥ 0")
			return
		}
		if seen[entry.ID] {
			s.writeError(w, http.StatusBadRequest, "duplicate rule id at index "+strconv.Itoa(i))
			return
		}
		seen[entry.ID] = true
	}

	ctx := r.Context()

	// Execute all updates in a single transaction.
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to begin transaction")
		return
	}
	defer tx.Rollback() //nolint:errcheck

	for _, entry := range body.Rules {
		res, err := tx.ExecContext(ctx,
			`UPDATE routing_rules SET priority = $1, updated_at = now() WHERE id = $2`,
			entry.Priority, entry.ID)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, "failed to update priority for "+entry.ID+": "+err.Error())
			return
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			s.writeError(w, http.StatusNotFound, "routing rule not found: "+entry.ID)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to commit transaction")
		return
	}

	// Return the updated rules in new priority order.
	rules, err := model.ListRoutingRules(ctx, s.db)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to reload rules")
		return
	}

	model.CreateAuditEntry(ctx, s.db, "routing_rule.reordered", actorEmail(r), "routing_rule", "", clientIP(r),
		map[string]any{"count": len(body.Rules)})

	s.writeJSON(w, http.StatusOK, map[string]any{"rules": rules, "updated": len(body.Rules)})
}
