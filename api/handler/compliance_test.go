package handler_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExportComplianceReportCSV_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/projects/00000000-0000-0000-0000-000000000000/compliance-report.csv", nil)
	req.SetPathValue("id", "00000000-0000-0000-0000-000000000000")
	w := httptest.NewRecorder()
	srv.ExportComplianceReportCSV(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestExportComplianceReportCSV_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/projects/"+proj.ID+"/compliance-report.csv", nil)
	req.SetPathValue("id", proj.ID)
	w := httptest.NewRecorder()
	srv.ExportComplianceReportCSV(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/csv", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Header().Get("Content-Disposition"), "attachment")
	assert.Contains(t, w.Header().Get("Content-Disposition"), ".csv")

	body := w.Body.String()
	assert.Contains(t, body, "section,metric,value")
	assert.Contains(t, body, "meta,project_id,"+proj.ID)
	assert.Contains(t, body, "studies,total,0")
	assert.Contains(t, body, "phi_detection,scanned,0")
	assert.Contains(t, body, "defacing,required,0")
	assert.Contains(t, body, "protocol_compliance,checked,0")
	assert.Contains(t, body, "exports,shares_created,0")
}

func TestExportComplianceReportCSV_ContainsProjectName(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/projects/"+proj.ID+"/compliance-report.csv", nil)
	req.SetPathValue("id", proj.ID)
	w := httptest.NewRecorder()
	srv.ExportComplianceReportCSV(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	// CSV should contain the project name in the meta section
	assert.True(t, strings.Contains(body, "meta,project_name,"))
	assert.True(t, strings.Contains(body, "meta,period_days,"))
}

func TestExportComplianceReportCSV_PHIFlagRate(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	// Create 2 studies: one clean, one flagged.
	s1 := testutil.CreateTestStudy(t, db, proj.ID)
	s2 := testutil.CreateTestStudy(t, db, proj.ID)
	_, err := db.Exec(`UPDATE studies SET phi_scan_status='clean' WHERE id=$1`, s1.ID)
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE studies SET phi_scan_status='flagged' WHERE id=$1`, s2.ID)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/projects/"+proj.ID+"/compliance-report.csv", nil)
	req.SetPathValue("id", proj.ID)
	w := httptest.NewRecorder()
	srv.ExportComplianceReportCSV(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "phi_detection,scanned,2")
	assert.Contains(t, body, "phi_detection,flagged,1")
	assert.Contains(t, body, "phi_detection,flag_rate_pct,50.00")
}
