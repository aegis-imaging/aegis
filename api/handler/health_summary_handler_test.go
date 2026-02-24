package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetSystemHealthSummary_OK(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/system/health-summary", nil)
	rr := httptest.NewRecorder()
	srv.GetSystemHealthSummary(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var result map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))

	// Top-level keys must be present.
	assert.NotEmpty(t, result["generated_at"])

	api := result["api"].(map[string]any)
	assert.Equal(t, "ok", api["status"])
	assert.Equal(t, "healthy", api["database"])
	assert.Equal(t, "healthy", api["storage"])

	pipeline := result["pipeline"].(map[string]any)
	_, hasReceived := pipeline["studies_received_24h"]
	_, hasApproved := pipeline["studies_approved_24h"]
	_, hasStuck := pipeline["studies_stuck"]
	assert.True(t, hasReceived)
	assert.True(t, hasApproved)
	assert.True(t, hasStuck)

	dimse := result["dimse"].(map[string]any)
	_, hasPending := dimse["pending_retries"]
	assert.True(t, hasPending)
}

func TestGetSystemHealthSummary_ServicesShape(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/system/health-summary", nil)
	rr := httptest.NewRecorder()
	srv.GetSystemHealthSummary(rr, req)

	var result map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))

	services, ok := result["services"].(map[string]any)
	require.True(t, ok)
	// All services should have a "status" field.
	for name, raw := range services {
		svc, ok := raw.(map[string]any)
		require.True(t, ok, "service %s should be a map", name)
		assert.NotEmpty(t, svc["status"], "service %s missing status", name)
	}
}
