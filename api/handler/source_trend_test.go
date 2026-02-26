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

func TestGetSourceTrend_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/stats/source-trend", nil)
	w := httptest.NewRecorder()
	srv.GetSourceTrend(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		PeriodDays int `json:"period_days"`
		Totals     struct {
			External int `json:"external"`
			Internal int `json:"internal"`
			Total    int `json:"total"`
		} `json:"totals"`
		Days []interface{} `json:"days"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 30, resp.PeriodDays)
	assert.Equal(t, 0, resp.Totals.Total)
	assert.Empty(t, resp.Days)
}

func TestGetSourceTrend_WithStudies(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	// CreateTestStudy defaults to source="external"
	testutil.CreateTestStudy(t, db, proj.ID)
	testutil.CreateTestStudy(t, db, proj.ID)
	s3 := testutil.CreateTestStudy(t, db, proj.ID)
	_, err := db.Exec(`UPDATE studies SET source='internal' WHERE id=$1`, s3.ID)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/stats/source-trend", nil)
	w := httptest.NewRecorder()
	srv.GetSourceTrend(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Totals struct {
			External int `json:"external"`
			Internal int `json:"internal"`
			Total    int `json:"total"`
		} `json:"totals"`
		Days []struct {
			Day      string `json:"day"`
			External int    `json:"external"`
			Internal int    `json:"internal"`
			Total    int    `json:"total"`
		} `json:"days"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 2, resp.Totals.External)
	assert.Equal(t, 1, resp.Totals.Internal)
	assert.Equal(t, 3, resp.Totals.Total)
	require.Len(t, resp.Days, 1)
	assert.Equal(t, 3, resp.Days[0].Total)
}

func TestGetSourceTrend_DaysParam(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/stats/source-trend?days=14", nil)
	w := httptest.NewRecorder()
	srv.GetSourceTrend(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		PeriodDays int `json:"period_days"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 14, resp.PeriodDays)
}

func TestGetSourceTrend_ProjectFilter(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	proj2 := testutil.CreateTestProject(t, db, "Other")

	testutil.CreateTestStudy(t, db, proj.ID)
	testutil.CreateTestStudy(t, db, proj2.ID)

	req := httptest.NewRequest(http.MethodGet, "/api/stats/source-trend?project_id="+proj.ID, nil)
	w := httptest.NewRecorder()
	srv.GetSourceTrend(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		ProjectID string `json:"project_id"`
		Totals    struct {
			Total int `json:"total"`
		} `json:"totals"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, proj.ID, resp.ProjectID)
	assert.Equal(t, 1, resp.Totals.Total)
}
