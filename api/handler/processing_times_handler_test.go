package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetProcessingTimes_Empty verifies an empty response when no audit events exist.
func TestGetProcessingTimes_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest("GET", "/api/stats/processing-times", nil)
	rr := httptest.NewRecorder()
	srv.GetProcessingTimes(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var resp struct {
		PeriodDays int                      `json:"period_days"`
		Stages     []map[string]interface{} `json:"stages"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, 30, resp.PeriodDays)
	// No audit events — stages list should be empty (nil marshals as null, coerce).
	assert.Empty(t, resp.Stages)
}

// TestGetProcessingTimes_WithData seeds paired triggered/complete audit events
// and verifies that the handler returns correct per-stage statistics.
func TestGetProcessingTimes_WithData(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)
	ctx := context.Background()

	// Seed a deface.triggered + deface.complete pair (~10 second gap).
	require.NoError(t, model.CreateAuditEntry(ctx, db,
		"deface.triggered", "system", "study", study.ID, "", nil))

	// Advance time by inserting the complete event 10 seconds later via SQL.
	_, err := db.ExecContext(ctx, `
		INSERT INTO audit_trail (action, actor, resource_type, resource_id, ip_address, created_at)
		VALUES ('deface.complete', 'system', 'study', $1, '', NOW() + INTERVAL '10 seconds')
	`, study.ID)
	require.NoError(t, err)

	// Also seed a phi_scan pair (~5 seconds) for a second study.
	study2 := testutil.CreateTestStudy(t, db, proj.ID)
	require.NoError(t, model.CreateAuditEntry(ctx, db,
		"phi_scan.triggered", "system", "study", study2.ID, "", nil))
	_, err = db.ExecContext(ctx, `
		INSERT INTO audit_trail (action, actor, resource_type, resource_id, ip_address, created_at)
		VALUES ('phi_scan.complete', 'system', 'study', $1, '', NOW() + INTERVAL '5 seconds')
	`, study2.ID)
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/api/stats/processing-times?days=30", nil)
	rr := httptest.NewRecorder()
	srv.GetProcessingTimes(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var resp struct {
		Stages []struct {
			Stage      string  `json:"stage"`
			Count      int     `json:"count"`
			AvgSeconds float64 `json:"avg_seconds"`
			P95Seconds float64 `json:"p95_seconds"`
		} `json:"stages"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))

	stageMap := make(map[string]struct {
		Count      int
		AvgSeconds float64
	})
	for _, s := range resp.Stages {
		stageMap[s.Stage] = struct {
			Count      int
			AvgSeconds float64
		}{s.Count, s.AvgSeconds}
	}

	// deface: 1 event, ~10 seconds average.
	def, ok := stageMap["deface"]
	require.True(t, ok, "expected deface stage in response")
	assert.Equal(t, 1, def.Count)
	assert.InDelta(t, 10.0, def.AvgSeconds, 1.5)

	// phi_scan: 1 event, ~5 seconds average.
	phi, ok := stageMap["phi_scan"]
	require.True(t, ok, "expected phi_scan stage in response")
	assert.Equal(t, 1, phi.Count)
	assert.InDelta(t, 5.0, phi.AvgSeconds, 1.5)
}

// TestGetProcessingTimes_ProjectFilter verifies that project_id scoping works.
func TestGetProcessingTimes_ProjectFilter(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)
	ctx := context.Background()

	// Seed one classification pair for this project.
	require.NoError(t, model.CreateAuditEntry(ctx, db,
		"classification.triggered", "system", "study", study.ID, "", nil))
	_, err := db.ExecContext(ctx, `
		INSERT INTO audit_trail (action, actor, resource_type, resource_id, ip_address, created_at)
		VALUES ('classification.complete', 'system', 'study', $1, '', NOW() + INTERVAL '3 seconds')
	`, study.ID)
	require.NoError(t, err)

	// Filter by this project — should see classification stage.
	req := httptest.NewRequest("GET", "/api/stats/processing-times?project_id="+proj.ID, nil)
	rr := httptest.NewRecorder()
	srv.GetProcessingTimes(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	var resp struct {
		Stages []struct {
			Stage string `json:"stage"`
			Count int    `json:"count"`
		} `json:"stages"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))

	var found bool
	for _, s := range resp.Stages {
		if s.Stage == "classification" {
			found = true
			assert.Equal(t, 1, s.Count)
		}
	}
	assert.True(t, found, "expected classification stage for project filter")

	// Filter by a random UUID — should return empty stages.
	req2 := httptest.NewRequest("GET",
		"/api/stats/processing-times?project_id=00000000-0000-0000-0000-000000000000", nil)
	rr2 := httptest.NewRecorder()
	srv.GetProcessingTimes(rr2, req2)
	require.Equal(t, http.StatusOK, rr2.Code)

	var resp2 struct {
		Stages []struct{} `json:"stages"`
	}
	require.NoError(t, json.NewDecoder(rr2.Body).Decode(&resp2))
	assert.Empty(t, resp2.Stages)
}
