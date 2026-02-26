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

func TestGetPhiTrend_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/stats/phi-trend", nil)
	w := httptest.NewRecorder()
	srv.GetPhiTrend(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var resp struct {
		PeriodDays  int    `json:"period_days"`
		GeneratedAt string `json:"generated_at"`
		Totals      struct {
			Scanned     int     `json:"scanned"`
			Flagged     int     `json:"flagged"`
			FlagRatePct float64 `json:"flag_rate_pct"`
		} `json:"totals"`
		Days []interface{} `json:"days"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 30, resp.PeriodDays)
	assert.NotEmpty(t, resp.GeneratedAt)
	assert.Equal(t, 0, resp.Totals.Scanned)
	assert.Equal(t, 0, resp.Totals.Flagged)
	assert.NotNil(t, resp.Days)
}

func TestGetPhiTrend_CustomDays(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/stats/phi-trend?days=7", nil)
	w := httptest.NewRecorder()
	srv.GetPhiTrend(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		PeriodDays int `json:"period_days"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 7, resp.PeriodDays)
}

func TestGetPhiTrend_InvalidDaysFallsBackToDefault(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/stats/phi-trend?days=0", nil)
	w := httptest.NewRecorder()
	srv.GetPhiTrend(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		PeriodDays int `json:"period_days"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 30, resp.PeriodDays)
}

func TestGetPhiTrend_WithScannedStudies(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	// Create 3 studies: 1 clean, 2 flagged
	s1 := testutil.CreateTestStudy(t, db, proj.ID)
	s2 := testutil.CreateTestStudy(t, db, proj.ID)
	s3 := testutil.CreateTestStudy(t, db, proj.ID)
	_, err := db.Exec(`UPDATE studies SET phi_scan_status='clean' WHERE id=$1`, s1.ID)
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE studies SET phi_scan_status='flagged' WHERE id=$1`, s2.ID)
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE studies SET phi_scan_status='flagged' WHERE id=$1`, s3.ID)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/stats/phi-trend?project_id="+proj.ID, nil)
	w := httptest.NewRecorder()
	srv.GetPhiTrend(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Totals struct {
			Scanned     int     `json:"scanned"`
			Flagged     int     `json:"flagged"`
			FlagRatePct float64 `json:"flag_rate_pct"`
		} `json:"totals"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 3, resp.Totals.Scanned)
	assert.Equal(t, 2, resp.Totals.Flagged)
	assert.InDelta(t, 66.67, resp.Totals.FlagRatePct, 0.01)
}

func TestGetPhiTrend_ProjectFilter(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	s := testutil.CreateTestStudy(t, db, proj.ID)
	_, err := db.Exec(`UPDATE studies SET phi_scan_status='clean' WHERE id=$1`, s.ID)
	require.NoError(t, err)

	// Project-scoped: should see 1 scan
	req := httptest.NewRequest(http.MethodGet, "/api/stats/phi-trend?project_id="+proj.ID, nil)
	w := httptest.NewRecorder()
	srv.GetPhiTrend(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Totals struct{ Scanned int `json:"scanned"` } `json:"totals"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 1, resp.Totals.Scanned)
}
