package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetProjectComplianceReport_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/projects/00000000-0000-0000-0000-000000000000/compliance-report", nil)
	req.SetPathValue("id", "00000000-0000-0000-0000-000000000000")
	w := httptest.NewRecorder()
	srv.GetProjectComplianceReport(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetProjectComplianceReport_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/projects/"+proj.ID+"/compliance-report", nil)
	req.SetPathValue("id", proj.ID)
	w := httptest.NewRecorder()
	srv.GetProjectComplianceReport(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var resp struct {
		ProjectID   string `json:"project_id"`
		PeriodDays  int    `json:"period_days"`
		GeneratedAt string `json:"generated_at"`
		Studies     struct {
			Total    int `json:"total"`
			Approved int `json:"approved"`
			Rejected int `json:"rejected"`
			Pending  int `json:"pending"`
		} `json:"studies"`
		PhiDetect struct {
			Scanned     int     `json:"scanned"`
			Flagged     int     `json:"flagged"`
			FlagRatePct float64 `json:"flag_rate_pct"`
		} `json:"phi_detection"`
		Defacing struct {
			Required  int     `json:"required"`
			Completed int     `json:"completed"`
			Failed    int     `json:"failed"`
			AvgQAScore float64 `json:"avg_qa_score"`
		} `json:"defacing"`
		Protocol struct {
			Checked         int `json:"checked"`
			Compliant       int `json:"compliant"`
			MinorDeviations int `json:"minor_deviations"`
			NonCompliant    int `json:"non_compliant"`
		} `json:"protocol_compliance"`
		Exports struct {
			SharesCreated    int `json:"shares_created"`
			SharesDownloaded int `json:"shares_downloaded"`
			TotalDownloads   int `json:"total_downloads"`
		} `json:"exports"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	assert.Equal(t, proj.ID, resp.ProjectID)
	assert.Equal(t, 30, resp.PeriodDays)
	assert.NotEmpty(t, resp.GeneratedAt)
	assert.Equal(t, 0, resp.Studies.Total)
	assert.Equal(t, 0, resp.PhiDetect.Scanned)
	assert.InDelta(t, 0.0, resp.PhiDetect.FlagRatePct, 0.001)
}

func TestGetProjectComplianceReport_WithStudies(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	study := testutil.CreateTestStudy(t, db, proj.ID)
	_, err := db.Exec(`UPDATE studies SET status='approved' WHERE id=$1`, study.ID)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/projects/"+proj.ID+"/compliance-report?days=30", nil)
	req.SetPathValue("id", proj.ID)
	w := httptest.NewRecorder()
	srv.GetProjectComplianceReport(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		PeriodDays int `json:"period_days"`
		Studies    struct {
			Total    int `json:"total"`
			Approved int `json:"approved"`
		} `json:"studies"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 30, resp.PeriodDays)
	assert.Equal(t, 1, resp.Studies.Total)
	assert.Equal(t, 1, resp.Studies.Approved)
}

func TestGetProjectComplianceReport_CustomDays(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/projects/"+proj.ID+"/compliance-report?days=7", nil)
	req.SetPathValue("id", proj.ID)
	w := httptest.NewRecorder()
	srv.GetProjectComplianceReport(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct{ PeriodDays int `json:"period_days"` }
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 7, resp.PeriodDays)
}

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
