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

func TestGetDestinationHealth_NeverTested(t *testing.T) {
	if testing.Short() {
		t.Log("integration test")
		t.Skip()
	}

	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	dest := testutil.CreateTestDestination(t, db, "GCP STOW-RS")

	req := httptest.NewRequest(http.MethodGet, "/api/destinations/"+dest.ID+"/health", nil)
	req.SetPathValue("id", dest.ID)
	w := httptest.NewRecorder()
	srv.GetDestinationHealth(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Summary struct {
			DestinationID   string  `json:"destination_id"`
			Status          string  `json:"status"`
			TestCount       int     `json:"test_count"`
			SuccessCount    int     `json:"success_count"`
			SuccessRate     float64 `json:"success_rate"`
		} `json:"summary"`
		RecentTests []any `json:"recent_tests"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, dest.ID, resp.Summary.DestinationID)
	assert.Equal(t, "unknown", resp.Summary.Status)
	assert.Equal(t, 0, resp.Summary.TestCount)
	assert.InDelta(t, -1.0, resp.Summary.SuccessRate, 0.001) // -1 = never tested
	assert.Empty(t, resp.RecentTests)
}

func TestGetDestinationHealth_WithTestEntries(t *testing.T) {
	if testing.Short() {
		t.Log("integration test")
		t.Skip()
	}

	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	dest := testutil.CreateTestDestination(t, db, "AWS STOW-RS")

	// Insert two successful test entries and one failure via audit_trail.
	for i, success := range []bool{true, true, false} {
		detail := map[string]any{"type": "dicomweb", "success": success, "latency_ms": float64(50 + i*10)}
		detailJSON, _ := json.Marshal(detail)
		_, err := db.Exec(`
			INSERT INTO audit_trail (action, actor, resource_type, resource_id, detail)
			VALUES ('destination.tested', 'ops@test.com', 'destination', $1, $2)`,
			dest.ID, detailJSON)
		require.NoError(t, err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/destinations/"+dest.ID+"/health?limit=10", nil)
	req.SetPathValue("id", dest.ID)
	w := httptest.NewRecorder()
	srv.GetDestinationHealth(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Summary struct {
			TestCount    int     `json:"test_count"`
			SuccessCount int     `json:"success_count"`
			FailureCount int     `json:"failure_count"`
			SuccessRate  float64 `json:"success_rate"`
			Status       string  `json:"status"`
		} `json:"summary"`
		RecentTests []any `json:"recent_tests"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 3, resp.Summary.TestCount)
	assert.Equal(t, 2, resp.Summary.SuccessCount)
	assert.Equal(t, 1, resp.Summary.FailureCount)
	assert.InDelta(t, 2.0/3.0, resp.Summary.SuccessRate, 0.01)
	assert.Equal(t, "degraded", resp.Summary.Status) // 67% < 90% threshold
	assert.Len(t, resp.RecentTests, 3)
}

func TestGetAllDestinationsHealth(t *testing.T) {
	if testing.Short() {
		t.Log("integration test")
		t.Skip()
	}

	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	testutil.CreateTestDestination(t, db, "Dest A")
	testutil.CreateTestDestination(t, db, "Dest B")

	req := httptest.NewRequest(http.MethodGet, "/api/destinations/health", nil)
	w := httptest.NewRecorder()
	srv.GetAllDestinationsHealth(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Destinations []struct {
			DestinationName string `json:"destination_name"`
			Status          string `json:"status"`
		} `json:"destinations"`
		Total int `json:"total"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.GreaterOrEqual(t, resp.Total, 2) // may be more from seed data
}

func TestGetDestinationHealth_NotFound(t *testing.T) {
	if testing.Short() {
		t.Log("integration test")
		t.Skip()
	}

	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/destinations/00000000-0000-0000-0000-000000000000/health", nil)
	req.SetPathValue("id", "00000000-0000-0000-0000-000000000000")
	w := httptest.NewRecorder()
	srv.GetDestinationHealth(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
