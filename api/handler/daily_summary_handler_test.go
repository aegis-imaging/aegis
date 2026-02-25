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

// ─── GetDailySummary ─────────────────────────────────────────────────────────

func TestGetDailySummary_DefaultHours(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/stats/daily-summary", nil)
	rr := httptest.NewRecorder()
	srv.GetDailySummary(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))

	// Required top-level fields.
	assert.EqualValues(t, 24, resp["period_hours"])
	assert.NotEmpty(t, resp["generated_at"])
	assert.NotEmpty(t, resp["period_started_at"])

	// Nested sections must be present.
	assert.NotNil(t, resp["ingestion"])
	assert.NotNil(t, resp["pipeline"])
	assert.NotNil(t, resp["routing"])
	assert.NotNil(t, resp["top_projects"])
	assert.NotNil(t, resp["recent_events"])
}

func TestGetDailySummary_CustomHours(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/stats/daily-summary?hours=48", nil)
	rr := httptest.NewRecorder()
	srv.GetDailySummary(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.EqualValues(t, 48, resp["period_hours"])
}

func TestGetDailySummary_HoursClampedToMax(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	// 999 is above the 168-hour max; handler clamps back to default 24.
	req := httptest.NewRequest(http.MethodGet, "/api/stats/daily-summary?hours=999", nil)
	rr := httptest.NewRecorder()
	srv.GetDailySummary(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	// Out-of-range value falls back to default 24.
	assert.EqualValues(t, 24, resp["period_hours"])
}

func TestGetDailySummary_WithStudies(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	testutil.CreateTestStudy(t, db, proj.ID)
	testutil.CreateTestStudy(t, db, proj.ID)

	req := httptest.NewRequest(http.MethodGet, "/api/stats/daily-summary?hours=24", nil)
	rr := httptest.NewRecorder()
	srv.GetDailySummary(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var resp struct {
		Ingestion struct {
			Received int `json:"received"`
		} `json:"ingestion"`
		TopProjects []struct {
			Slug     string `json:"slug"`
			Received int    `json:"received"`
		} `json:"top_projects"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))

	assert.GreaterOrEqual(t, resp.Ingestion.Received, 2)
	assert.NotEmpty(t, resp.TopProjects)
}

func TestGetDailySummary_IngestionSectionsPresent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/stats/daily-summary", nil)
	rr := httptest.NewRecorder()
	srv.GetDailySummary(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	// Fully decode into a typed struct to verify all nested fields are present.
	var resp struct {
		PeriodHours     int    `json:"period_hours"`
		GeneratedAt     string `json:"generated_at"`
		PeriodStartedAt string `json:"period_started_at"`
		Ingestion       struct {
			Received int `json:"received"`
			Approved int `json:"approved"`
			Rejected int `json:"rejected"`
			Stuck    int `json:"stuck"`
		} `json:"ingestion"`
		Pipeline struct {
			PendingReview int `json:"pending_review"`
			InProcessing  int `json:"in_processing"`
			Failed        int `json:"failed"`
		} `json:"pipeline"`
		Routing struct {
			Attempts    int     `json:"attempts"`
			Successful  int     `json:"successful"`
			Failed      int     `json:"failed"`
			SuccessRate float64 `json:"success_rate"`
		} `json:"routing"`
		TopProjects  []any `json:"top_projects"`
		RecentEvents []any `json:"recent_events"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))

	assert.Equal(t, 24, resp.PeriodHours)
	assert.NotEmpty(t, resp.GeneratedAt)
	assert.NotEmpty(t, resp.PeriodStartedAt)
	// Zero values are valid (empty DB) — just assert the structure decoded cleanly.
	assert.NotNil(t, resp.TopProjects)
	assert.NotNil(t, resp.RecentEvents)
}
