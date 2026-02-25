package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetRoutingRuleStats_NoLogs verifies the response when routing_log is empty —
// all rules appear in unused_rules, by_rule is empty.
func TestGetRoutingRuleStats_NoLogs(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	// Seed one routing rule but no log entries.
	testutil.CreateTestRoutingRule(t, db, "Test Rule", "require_qc_check")

	req := httptest.NewRequest("GET", "/api/stats/routing-rules", nil)
	rr := httptest.NewRecorder()
	srv.GetRoutingRuleStats(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var resp struct {
		PeriodDays  int `json:"period_days"`
		TotalHits   int `json:"total_hits"`
		ActiveRules int `json:"active_rules"`
		ByRule      []struct {
			RuleName string `json:"rule_name"`
			HitCount int    `json:"hit_count"`
		} `json:"by_rule"`
		UnusedRules []struct {
			RuleName string `json:"rule_name"`
		} `json:"unused_rules"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, 30, resp.PeriodDays)
	assert.Equal(t, 0, resp.TotalHits)
	assert.Equal(t, 0, resp.ActiveRules)
	assert.Len(t, resp.ByRule, 0)
	// The seeded rule should appear in unused_rules.
	found := false
	for _, u := range resp.UnusedRules {
		if u.RuleName == "Test Rule" {
			found = true
		}
	}
	assert.True(t, found, "seeded rule should appear in unused_rules")
}

// TestGetRoutingRuleStats_WithHits seeds routing_log entries for a rule and verifies
// hit counts and that rules without entries appear in unused_rules.
func TestGetRoutingRuleStats_WithHits(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)
	rule := testutil.CreateTestRoutingRule(t, db, "Head Deface Rule", "require_defacing")
	_ = testutil.CreateTestRoutingRule(t, db, "Unused Rule", "require_qc_check")

	// Seed 5 log entries for rule (no destination — action is require_defacing).
	for i := 0; i < 5; i++ {
		_, err := db.ExecContext(context.Background(), `
			INSERT INTO routing_log (study_id, rule_id, action, destination_id, outcome)
			VALUES ($1, $2, 'require_defacing', NULL, 'defacing_required set')`,
			study.ID, rule.ID)
		require.NoError(t, err)
	}

	req := httptest.NewRequest("GET", "/api/stats/routing-rules?days=30", nil)
	rr := httptest.NewRecorder()
	srv.GetRoutingRuleStats(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var resp struct {
		TotalHits   int `json:"total_hits"`
		ActiveRules int `json:"active_rules"`
		ByRule      []struct {
			RuleID   string `json:"rule_id"`
			RuleName string `json:"rule_name"`
			HitCount int    `json:"hit_count"`
		} `json:"by_rule"`
		UnusedRules []struct {
			RuleName string `json:"rule_name"`
		} `json:"unused_rules"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))

	assert.Equal(t, 5, resp.TotalHits)
	assert.Equal(t, 1, resp.ActiveRules)
	require.Len(t, resp.ByRule, 1)
	assert.Equal(t, "Head Deface Rule", resp.ByRule[0].RuleName)
	assert.Equal(t, 5, resp.ByRule[0].HitCount)

	// "Unused Rule" should be in unused_rules.
	found := false
	for _, u := range resp.UnusedRules {
		if u.RuleName == "Unused Rule" {
			found = true
		}
	}
	assert.True(t, found, "unused rule should appear in unused_rules")
}
