package handler

import (
	"encoding/json"
	"net/http"

	"github.com/aegis-imaging/aegis/api/model"
)

type bulkToggleRequest struct {
	RuleIDs []string `json:"rule_ids"`
	Enabled bool     `json:"enabled"`
}

type bulkToggleResponse struct {
	Updated int      `json:"updated"`
	RuleIDs []string `json:"rule_ids"`
}

// BulkToggleRoutingRules enables or disables multiple routing rules atomically.
//
// POST /api/routing-rules/bulk-toggle
// Body: {"rule_ids": ["<uuid>", ...], "enabled": true|false}
//
// Returns the count of updated rules and their IDs.
// Up to 200 rule IDs per call. Unknown IDs are silently ignored (no 404).
func (s *Server) BulkToggleRoutingRules(w http.ResponseWriter, r *http.Request) {
	var req bulkToggleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if len(req.RuleIDs) == 0 {
		s.writeError(w, http.StatusBadRequest, "rule_ids must be a non-empty array")
		return
	}
	if len(req.RuleIDs) > 200 {
		s.writeError(w, http.StatusBadRequest, "rule_ids must not exceed 200 entries")
		return
	}
	for _, id := range req.RuleIDs {
		if !isValidUUID(id) {
			s.writeError(w, http.StatusBadRequest, "invalid rule_id: "+id)
			return
		}
	}

	ctx := r.Context()

	// Build array literal for PostgreSQL ANY($1::uuid[]).
	ids := make([]interface{}, len(req.RuleIDs))
	placeholders := make([]string, len(req.RuleIDs))
	for i, id := range req.RuleIDs {
		ids[i] = id
		placeholders[i] = "$" + itoa(i+2)
	}

	// Use pq-style ANY with a cast to avoid the interface{} expansion issue —
	// build a plain UUID array arg using lib/pq or use individual placeholders.
	// We collect matched IDs via RETURNING to report exactly what was changed.
	rows, err := s.db.QueryContext(ctx,
		buildBulkToggleQuery(len(req.RuleIDs), req.Enabled),
		buildBulkToggleArgs(req.RuleIDs, req.Enabled)...,
	)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "toggle query failed")
		return
	}
	defer rows.Close()

	updated := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			updated = append(updated, id)
		}
	}

	model.CreateAuditEntry(ctx, s.db, "routing_rule.bulk_toggled", actorEmail(r), "routing_rule", "", clientIP(r),
		map[string]any{"enabled": req.Enabled, "count": len(updated), "rule_ids": req.RuleIDs},
	)

	s.writeJSON(w, http.StatusOK, bulkToggleResponse{
		Updated: len(updated),
		RuleIDs: updated,
	})
}

// buildBulkToggleQuery constructs a parameterized UPDATE … WHERE id = ANY(…) RETURNING id.
// $1 = enabled, $2..$N = rule UUIDs.
func buildBulkToggleQuery(n int, _ bool) string {
	// We use a fixed-length placeholder expansion compatible with database/sql.
	// PostgreSQL accepts: UPDATE … WHERE id IN ($2,$3,…)
	if n == 0 {
		return ""
	}
	q := "UPDATE routing_rules SET enabled = $1, updated_at = now() WHERE id IN ("
	for i := 0; i < n; i++ {
		if i > 0 {
			q += ","
		}
		q += "$" + itoa(i+2)
	}
	q += ") RETURNING id"
	return q
}

// buildBulkToggleArgs assembles the argument slice: [enabled, id1, id2, ...].
func buildBulkToggleArgs(ids []string, enabled bool) []interface{} {
	args := make([]interface{}, 1+len(ids))
	args[0] = enabled
	for i, id := range ids {
		args[i+1] = id
	}
	return args
}

// itoa converts an int to its decimal string representation (avoids importing strconv).
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := [20]byte{}
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[pos:])
}
