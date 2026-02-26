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

func TestGetProtocolTrend_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/stats/protocol-trend", nil)
	w := httptest.NewRecorder()
	srv.GetProtocolTrend(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var resp struct {
		GeneratedAt string `json:"generated_at"`
		PeriodDays  int    `json:"period_days"`
		ProjectID   string `json:"project_id,omitempty"`
		Totals      struct {
			Checked         int     `json:"checked"`
			Compliant       int     `json:"compliant"`
			MinorDeviations int     `json:"minor_deviations"`
			NonCompliant    int     `json:"non_compliant"`
			CompliancePct   float64 `json:"compliance_pct"`
		} `json:"totals"`
		Days []struct {
			Day           string  `json:"day"`
			Checked       int     `json:"checked"`
			Compliant     int     `json:"compliant"`
			CompliancePct float64 `json:"compliance_pct"`
		} `json:"days"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	assert.NotEmpty(t, resp.GeneratedAt)
	assert.Equal(t, 30, resp.PeriodDays) // default
	assert.Empty(t, resp.ProjectID)      // no project filter
	assert.Equal(t, 0, resp.Totals.Checked)
	assert.InDelta(t, -1.0, resp.Totals.CompliancePct, 0.001) // -1 when no checks
	assert.Empty(t, resp.Days)
}

func TestGetProtocolTrend_CustomDays(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/stats/protocol-trend?days=7", nil)
	w := httptest.NewRecorder()
	srv.GetProtocolTrend(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct{ PeriodDays int `json:"period_days"` }
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 7, resp.PeriodDays)
}

func TestGetProtocolTrend_DaysOutOfRange(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	// days=0 is invalid — should fall back to default 30
	req := httptest.NewRequest(http.MethodGet, "/api/stats/protocol-trend?days=0", nil)
	w := httptest.NewRecorder()
	srv.GetProtocolTrend(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct{ PeriodDays int `json:"period_days"` }
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 30, resp.PeriodDays) // falls back to default
}

func TestGetProtocolTrend_WithCompliantStudy(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	study := testutil.CreateTestStudy(t, db, proj.ID)
	_, err := db.Exec(
		`UPDATE studies SET protocol_required=true, protocol_status='compliant' WHERE id=$1`,
		study.ID)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/stats/protocol-trend?days=30", nil)
	w := httptest.NewRecorder()
	srv.GetProtocolTrend(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Totals struct {
			Checked       int     `json:"checked"`
			Compliant     int     `json:"compliant"`
			CompliancePct float64 `json:"compliance_pct"`
		} `json:"totals"`
		Days []interface{} `json:"days"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 1, resp.Totals.Checked)
	assert.Equal(t, 1, resp.Totals.Compliant)
	assert.InDelta(t, 100.0, resp.Totals.CompliancePct, 0.001)
	assert.NotEmpty(t, resp.Days) // at least one day row
}

func TestGetProtocolTrend_ProjectFilter(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	// Add a compliant study for this project
	study := testutil.CreateTestStudy(t, db, proj.ID)
	_, err := db.Exec(
		`UPDATE studies SET protocol_required=true, protocol_status='compliant' WHERE id=$1`,
		study.ID)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/stats/protocol-trend?project_id="+proj.ID, nil)
	w := httptest.NewRecorder()
	srv.GetProtocolTrend(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		ProjectID string `json:"project_id"`
		Totals    struct {
			Checked int `json:"checked"`
		} `json:"totals"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, proj.ID, resp.ProjectID)
	assert.Equal(t, 1, resp.Totals.Checked)
}

func TestGetProtocolTrend_MixedStatuses(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	// compliant, minor_deviations, non_compliant
	s1 := testutil.CreateTestStudy(t, db, proj.ID)
	s2 := testutil.CreateTestStudy(t, db, proj.ID)
	s3 := testutil.CreateTestStudy(t, db, proj.ID)

	_, err := db.Exec(`UPDATE studies SET protocol_required=true, protocol_status='compliant' WHERE id=$1`, s1.ID)
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE studies SET protocol_required=true, protocol_status='minor_deviations' WHERE id=$1`, s2.ID)
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE studies SET protocol_required=true, protocol_status='non_compliant' WHERE id=$1`, s3.ID)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/stats/protocol-trend?days=30&project_id="+proj.ID, nil)
	w := httptest.NewRecorder()
	srv.GetProtocolTrend(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Totals struct {
			Checked         int     `json:"checked"`
			Compliant       int     `json:"compliant"`
			MinorDeviations int     `json:"minor_deviations"`
			NonCompliant    int     `json:"non_compliant"`
			CompliancePct   float64 `json:"compliance_pct"`
		} `json:"totals"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 3, resp.Totals.Checked)
	assert.Equal(t, 1, resp.Totals.Compliant)
	assert.Equal(t, 1, resp.Totals.MinorDeviations)
	assert.Equal(t, 1, resp.Totals.NonCompliant)
	// compliance_pct = 1/3 * 100 ≈ 33.33
	assert.InDelta(t, 33.33, resp.Totals.CompliancePct, 0.1)
}
