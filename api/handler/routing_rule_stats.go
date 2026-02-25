package handler

import (
	"net/http"
	"strconv"
	"time"
)

// ruleHitEntry is one routing rule's analytics entry.
type ruleHitEntry struct {
	RuleID          string  `json:"rule_id"`
	RuleName        string  `json:"rule_name"`
	RuleAction      string  `json:"rule_action"`
	RuleEnabled     bool    `json:"rule_enabled"`
	DestinationID   *string `json:"destination_id"`
	DestinationName *string `json:"destination_name"`
	HitCount        int     `json:"hit_count"`
	LastMatchedAt   *string `json:"last_matched_at"`
	FirstMatchedAt  *string `json:"first_matched_at"`
}

// unusedRuleEntry is a routing rule that had zero log entries in the period.
type unusedRuleEntry struct {
	RuleID      string  `json:"rule_id"`
	RuleName    string  `json:"rule_name"`
	RuleAction  string  `json:"rule_action"`
	RuleEnabled bool    `json:"rule_enabled"`
	DestinationID *string `json:"destination_id"`
}

// routingRuleStatsResponse is the payload for GetRoutingRuleStats.
type routingRuleStatsResponse struct {
	PeriodDays   int               `json:"period_days"`
	GeneratedAt  string            `json:"generated_at"`
	TotalHits    int               `json:"total_hits"`
	ActiveRules  int               `json:"active_rules"`
	UnusedRules  []unusedRuleEntry `json:"unused_rules"`
	ByRule       []ruleHitEntry    `json:"by_rule"`
}

// GetRoutingRuleStats returns per-rule hit counts derived from the routing_log table.
//
// GET /api/stats/routing-rules?days=30
func (s *Server) GetRoutingRuleStats(w http.ResponseWriter, r *http.Request) {
	days := 30
	if d := r.URL.Query().Get("days"); d != "" {
		if n, err := strconv.Atoi(d); err == nil && n > 0 && n <= 365 {
			days = n
		}
	}
	since := time.Now().UTC().AddDate(0, 0, -days)

	// Rules that have log entries in the period — ordered by hit_count DESC.
	rows, err := s.db.QueryContext(r.Context(), `
		SELECT
		  rl.rule_id,
		  rr.name,
		  rr.action,
		  rr.enabled,
		  d.id,
		  d.name,
		  COUNT(*)               AS hit_count,
		  MAX(rl.created_at)     AS last_matched_at,
		  MIN(rl.created_at)     AS first_matched_at
		FROM routing_log rl
		JOIN routing_rules rr ON rr.id = rl.rule_id
		LEFT JOIN destinations d ON d.id = rr.destination_id
		WHERE rl.created_at >= $1
		GROUP BY rl.rule_id, rr.name, rr.action, rr.enabled, d.id, d.name
		ORDER BY hit_count DESC`,
		since)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "routing rule stats query failed")
		return
	}
	defer rows.Close()

	byRule := []ruleHitEntry{}
	totalHits := 0
	seenRuleIDs := map[string]struct{}{}
	for rows.Next() {
		var e ruleHitEntry
		var lastAt, firstAt *time.Time
		var destID, destName *string
		if err := rows.Scan(&e.RuleID, &e.RuleName, &e.RuleAction, &e.RuleEnabled,
			&destID, &destName, &e.HitCount, &lastAt, &firstAt); err != nil {
			continue
		}
		if lastAt != nil {
			ts := lastAt.UTC().Format(time.RFC3339)
			e.LastMatchedAt = &ts
		}
		if firstAt != nil {
			ts := firstAt.UTC().Format(time.RFC3339)
			e.FirstMatchedAt = &ts
		}
		e.DestinationID = destID
		e.DestinationName = destName
		totalHits += e.HitCount
		seenRuleIDs[e.RuleID] = struct{}{}
		byRule = append(byRule, e)
	}

	// Rules that exist but had zero hits in the period.
	unusedRows, err := s.db.QueryContext(r.Context(), `
		SELECT rr.id, rr.name, rr.action, rr.enabled, d.id
		FROM routing_rules rr
		LEFT JOIN destinations d ON d.id = rr.destination_id
		ORDER BY rr.priority ASC, rr.name ASC`)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "routing rules query failed")
		return
	}
	defer unusedRows.Close()

	unused := []unusedRuleEntry{}
	for unusedRows.Next() {
		var u unusedRuleEntry
		var destID *string
		if err := unusedRows.Scan(&u.RuleID, &u.RuleName, &u.RuleAction, &u.RuleEnabled, &destID); err != nil {
			continue
		}
		if _, seen := seenRuleIDs[u.RuleID]; !seen {
			u.DestinationID = destID
			unused = append(unused, u)
		}
	}

	s.writeJSON(w, http.StatusOK, routingRuleStatsResponse{
		PeriodDays:  days,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		TotalHits:   totalHits,
		ActiveRules: len(byRule),
		ByRule:      byRule,
		UnusedRules: unused,
	})
}
