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

func TestGetProjectComplianceReport_NotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/projects/no-such-id/compliance-report", nil)
	req.SetPathValue("id", "no-such-id")
	rr := httptest.NewRecorder()
	srv.GetProjectComplianceReport(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestGetProjectComplianceReport_Empty(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/projects/"+proj.ID+"/compliance-report", nil)
	req.SetPathValue("id", proj.ID)
	rr := httptest.NewRecorder()
	srv.GetProjectComplianceReport(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var report map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&report))

	assert.Equal(t, proj.ID, report["project_id"])
	assert.NotEmpty(t, report["generated_at"])
	assert.Equal(t, float64(30), report["period_days"])

	studies := report["studies"].(map[string]any)
	assert.Equal(t, float64(0), studies["total"])
	assert.Equal(t, float64(0), studies["approved"])
}

func TestGetProjectComplianceReport_WithStudies(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	// Create a couple of studies.
	testutil.CreateTestStudy(t, db, proj.ID)
	testutil.CreateTestStudy(t, db, proj.ID)

	req := httptest.NewRequest(http.MethodGet, "/api/projects/"+proj.ID+"/compliance-report?days=7", nil)
	req.SetPathValue("id", proj.ID)
	rr := httptest.NewRecorder()
	srv.GetProjectComplianceReport(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var report map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&report))

	assert.Equal(t, float64(7), report["period_days"])
	studies := report["studies"].(map[string]any)
	assert.Equal(t, float64(2), studies["total"])
}

func TestGetProjectComplianceReport_DefaultDays(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/projects/"+proj.ID+"/compliance-report", nil)
	req.SetPathValue("id", proj.ID)
	rr := httptest.NewRecorder()
	srv.GetProjectComplianceReport(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var report map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&report))
	assert.Equal(t, float64(30), report["period_days"])
}
