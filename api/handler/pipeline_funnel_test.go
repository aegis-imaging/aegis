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

// TestGetPipelineFunnel_Empty verifies zero counts and pct values on an empty database.
func TestGetPipelineFunnel_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest("GET", "/api/stats/pipeline-funnel", nil)
	rr := httptest.NewRecorder()
	srv.GetPipelineFunnel(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var resp struct {
		PeriodDays int `json:"period_days"`
		Funnel     []struct {
			Stage      string  `json:"stage"`
			Count      int     `json:"count"`
			PctOfTotal float64 `json:"pct_of_total"`
		} `json:"funnel"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, 30, resp.PeriodDays)
	require.Len(t, resp.Funnel, 8)
	for _, s := range resp.Funnel {
		assert.Equal(t, 0, s.Count)
		assert.Equal(t, 0.0, s.PctOfTotal)
	}
}

// TestGetPipelineFunnel_WithStudies verifies funnel counts with a seeded study.
func TestGetPipelineFunnel_WithStudies(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	// Mark the study as classified and phi_scanned (clean).
	_, err := db.ExecContext(context.Background(), `
		UPDATE studies
		SET classification_status = 'classified',
		    phi_scan_required = true,
		    phi_scan_status = 'clean'
		WHERE id = $1`, study.ID)
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/api/stats/pipeline-funnel?project_id="+proj.ID, nil)
	rr := httptest.NewRecorder()
	srv.GetPipelineFunnel(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var resp struct {
		ProjectID string `json:"project_id"`
		Funnel    []struct {
			Stage      string  `json:"stage"`
			Count      int     `json:"count"`
			PctOfTotal float64 `json:"pct_of_total"`
		} `json:"funnel"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, proj.ID, resp.ProjectID)

	stageMap := map[string]int{}
	for _, f := range resp.Funnel {
		stageMap[f.Stage] = f.Count
	}
	assert.Equal(t, 1, stageMap["received"])
	assert.Equal(t, 1, stageMap["classified"])
	assert.Equal(t, 1, stageMap["phi_scanned"])
	assert.Equal(t, 0, stageMap["approved"])
}
