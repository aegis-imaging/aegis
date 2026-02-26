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

func TestGetModalityTrend_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/stats/modality-trend", nil)
	w := httptest.NewRecorder()
	srv.GetModalityTrend(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		PeriodDays int                      `json:"period_days"`
		Totals     map[string]int           `json:"totals"`
		Days       []interface{}            `json:"days"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 30, resp.PeriodDays)
	assert.Empty(t, resp.Totals)
	assert.Empty(t, resp.Days)
}

func TestGetModalityTrend_WithStudies(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	// Create 2 MRI and 1 CT study
	testutil.CreateTestStudy(t, db, proj.ID)
	testutil.CreateTestStudy(t, db, proj.ID)
	s3 := testutil.CreateTestStudy(t, db, proj.ID)
	_, err := db.Exec(`UPDATE studies SET modality='CT' WHERE id=$1`, s3.ID)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/stats/modality-trend", nil)
	w := httptest.NewRecorder()
	srv.GetModalityTrend(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		PeriodDays int            `json:"period_days"`
		Totals     map[string]int `json:"totals"`
		Days       []struct {
			Day    string         `json:"day"`
			Counts map[string]int `json:"counts"`
			Total  int            `json:"total"`
		} `json:"days"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 2, resp.Totals["MRI"])
	assert.Equal(t, 1, resp.Totals["CT"])
	require.Len(t, resp.Days, 1) // all created today
	assert.Equal(t, 3, resp.Days[0].Total)
}

func TestGetModalityTrend_DaysParam(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/stats/modality-trend?days=7", nil)
	w := httptest.NewRecorder()
	srv.GetModalityTrend(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		PeriodDays int `json:"period_days"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 7, resp.PeriodDays)
}

func TestGetModalityTrend_ProjectFilter(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	proj2 := testutil.CreateTestProject(t, db, "Other")

	testutil.CreateTestStudy(t, db, proj.ID)
	testutil.CreateTestStudy(t, db, proj2.ID)

	req := httptest.NewRequest(http.MethodGet, "/api/stats/modality-trend?project_id="+proj.ID, nil)
	w := httptest.NewRecorder()
	srv.GetModalityTrend(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		ProjectID string         `json:"project_id"`
		Totals    map[string]int `json:"totals"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, proj.ID, resp.ProjectID)
	assert.Equal(t, 1, resp.Totals["MRI"])
}
