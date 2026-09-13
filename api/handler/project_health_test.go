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

func TestGetProjectHealth_Empty(t *testing.T) {
	if testing.Short() {
		t.Log("integration test")
		t.Skip()
	}

	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/stats/project-health", nil)
	w := httptest.NewRecorder()
	srv.GetProjectHealth(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		PeriodDays  int    `json:"period_days"`
		GeneratedAt string `json:"generated_at"`
		StuckCount  int    `json:"stuck_count"`
		Routing     struct {
			Attempts int `json:"attempts"`
		} `json:"routing"`
		Funnel []struct {
			Stage string `json:"stage"`
			Count int    `json:"count"`
		} `json:"funnel"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 30, resp.PeriodDays)
	assert.NotEmpty(t, resp.GeneratedAt)
	assert.Equal(t, 0, resp.StuckCount)
	assert.Equal(t, 0, resp.Routing.Attempts)
	require.Len(t, resp.Funnel, 8)
	assert.Equal(t, "received", resp.Funnel[0].Stage)
	assert.Equal(t, 0, resp.Funnel[0].Count)
}

func TestGetProjectHealth_WithStudy(t *testing.T) {
	if testing.Short() {
		t.Log("integration test")
		t.Skip()
	}

	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	study := testutil.CreateTestStudy(t, db, proj.ID)

	// Mark classification complete.
	_, err := db.Exec(`UPDATE studies SET classification_status = 'classified' WHERE id = $1`, study.ID)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/stats/project-health?project_id="+proj.ID, nil)
	w := httptest.NewRecorder()
	srv.GetProjectHealth(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		ProjectID  string `json:"project_id"`
		StuckCount int    `json:"stuck_count"`
		Funnel     []struct {
			Stage      string  `json:"stage"`
			Count      int     `json:"count"`
			PctOfTotal float64 `json:"pct_of_total"`
		} `json:"funnel"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, proj.ID, resp.ProjectID)
	assert.Equal(t, 0, resp.StuckCount)
	require.Len(t, resp.Funnel, 8)

	// received = 1, classified = 1
	assert.Equal(t, 1, resp.Funnel[0].Count) // received
	assert.Equal(t, 1, resp.Funnel[1].Count) // classified
	assert.InDelta(t, 100.0, resp.Funnel[1].PctOfTotal, 0.01)

	// Not yet approved
	assert.Equal(t, "approved", resp.Funnel[6].Stage)
	assert.Equal(t, 0, resp.Funnel[6].Count)
}
