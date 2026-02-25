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

// TestGetRoutingStats_Empty verifies an empty/zero response when the routing_log is empty.
func TestGetRoutingStats_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest("GET", "/api/stats/routing", nil)
	rr := httptest.NewRecorder()
	srv.GetRoutingStats(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var resp struct {
		PeriodDays    int `json:"period_days"`
		Totals        struct {
			Attempts   int     `json:"attempts"`
			Successful int     `json:"successful"`
			SuccessRate float64 `json:"success_rate"`
		} `json:"totals"`
		ByDestination []interface{} `json:"by_destination"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, 30, resp.PeriodDays)
	assert.Equal(t, 0, resp.Totals.Attempts)
	assert.Equal(t, 0.0, resp.Totals.SuccessRate)
	assert.Empty(t, resp.ByDestination)
}

// TestGetDestinationStats_WithData seeds routing_log entries for a destination
// and verifies success/failure counts are returned correctly.
func TestGetDestinationStats_WithData(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	ctx := context.Background()

	// Create a DICOMweb destination.
	dest := testutil.CreateTestDestination(t, db, "Test AWS Dest")

	// Create a study for routing_log foreign key.
	study := testutil.CreateTestStudy(t, db, proj.ID)

	// Create a routing rule.
	rule := testutil.CreateTestRoutingRule(t, db, "route to aws", "route_to")

	// Seed routing_log: 3 successes + 1 failure.
	seedRoutingLog := func(outcome string) {
		_, err := db.ExecContext(ctx, `
			INSERT INTO routing_log (study_id, rule_id, action, destination_id, outcome)
			VALUES ($1, $2, 'route_to', $3, $4)
		`, study.ID, rule.ID, dest.ID, outcome)
		require.NoError(t, err)
	}

	seedRoutingLog("forwarding dispatched")
	seedRoutingLog("forwarding dispatched")
	seedRoutingLog("forwarding dispatched")
	seedRoutingLog("error: connection refused to host")

	// Test per-destination stats.
	req := httptest.NewRequest("GET", "/api/destinations/"+dest.ID+"/stats", nil)
	req.SetPathValue("id", dest.ID)
	rr := httptest.NewRecorder()
	srv.GetDestinationStats(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var resp struct {
		DestinationID string  `json:"destination_id"`
		TotalAttempts int     `json:"total_attempts"`
		Successful    int     `json:"successful"`
		Failed        int     `json:"failed"`
		SuccessRate   float64 `json:"success_rate"`
		RecentErrors  []struct {
			Outcome string `json:"outcome"`
		} `json:"recent_errors"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))

	assert.Equal(t, dest.ID, resp.DestinationID)
	assert.Equal(t, 4, resp.TotalAttempts)
	assert.Equal(t, 3, resp.Successful)
	assert.Equal(t, 1, resp.Failed)
	assert.InDelta(t, 0.75, resp.SuccessRate, 0.001)
	require.Len(t, resp.RecentErrors, 1)
	assert.Equal(t, "error: connection refused to host", resp.RecentErrors[0].Outcome)
}

// TestGetRoutingStats_ByDestination verifies the overview includes per-destination breakdown.
func TestGetRoutingStats_ByDestination(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	ctx := context.Background()

	dest := testutil.CreateTestDestination(t, db, "GCP Dest")
	study := testutil.CreateTestStudy(t, db, proj.ID)
	rule := testutil.CreateTestRoutingRule(t, db, "route to gcp", "route_to")

	// Seed 2 successful routing attempts.
	for i := 0; i < 2; i++ {
		_, err := db.ExecContext(ctx, `
			INSERT INTO routing_log (study_id, rule_id, action, destination_id, outcome)
			VALUES ($1, $2, 'route_to', $3, 'forwarding dispatched')
		`, study.ID, rule.ID, dest.ID)
		require.NoError(t, err)
	}

	req := httptest.NewRequest("GET", "/api/stats/routing", nil)
	rr := httptest.NewRecorder()
	srv.GetRoutingStats(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var resp struct {
		Totals struct {
			Attempts   int     `json:"attempts"`
			Successful int     `json:"successful"`
			SuccessRate float64 `json:"success_rate"`
		} `json:"totals"`
		ByDestination []struct {
			DestinationID   string  `json:"destination_id"`
			DestinationName string  `json:"destination_name"`
			Attempts        int     `json:"attempts"`
			Successful      int     `json:"successful"`
			SuccessRate     float64 `json:"success_rate"`
		} `json:"by_destination"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))

	assert.Equal(t, 2, resp.Totals.Attempts)
	assert.Equal(t, 2, resp.Totals.Successful)
	assert.InDelta(t, 1.0, resp.Totals.SuccessRate, 0.001)

	require.Len(t, resp.ByDestination, 1)
	assert.Equal(t, dest.ID, resp.ByDestination[0].DestinationID)
	assert.Equal(t, "GCP Dest", resp.ByDestination[0].DestinationName)
	assert.Equal(t, 2, resp.ByDestination[0].Attempts)
	assert.InDelta(t, 1.0, resp.ByDestination[0].SuccessRate, 0.001)
}
