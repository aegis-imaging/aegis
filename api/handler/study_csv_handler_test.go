package handler_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
)

func TestExportStudiesCSV_OK(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	testutil.CreateTestStudy(t, db, proj.ID)

	req := httptest.NewRequest("GET", "/api/studies.csv", nil)
	rr := httptest.NewRecorder()
	srv.ExportStudiesCSV(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Header().Get("Content-Disposition"), "studies.csv")
	body := rr.Body.String()
	// Should have a header row and at least one data row
	lines := strings.Split(strings.TrimSpace(body), "\n")
	assert.GreaterOrEqual(t, len(lines), 2, "expected header + at least one data row")
}

func TestExportStudiesCSV_Empty(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest("GET", "/api/studies.csv", nil)
	rr := httptest.NewRecorder()
	srv.ExportStudiesCSV(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	body := rr.Body.String()
	// Should only have the header row
	lines := strings.Split(strings.TrimSpace(body), "\n")
	assert.Len(t, lines, 1, "expected only header row when no studies exist")
}

func TestExportStudiesCSV_WithStatusFilter(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	testutil.CreateTestStudy(t, db, proj.ID)

	req := httptest.NewRequest("GET", "/api/studies.csv?status=approved", nil)
	rr := httptest.NewRecorder()
	srv.ExportStudiesCSV(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	// No approved studies — only header row
	body := rr.Body.String()
	lines := strings.Split(strings.TrimSpace(body), "\n")
	assert.Len(t, lines, 1, "expected only header row for approved filter with no approved studies")
}

func TestExportStudiesCSV_WithLabelFilter(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	testutil.CreateTestStudy(t, db, proj.ID)

	// Filter by a label that doesn't exist — expect only header row
	req := httptest.NewRequest("GET", "/api/studies.csv?label=nonexistent-label", nil)
	rr := httptest.NewRecorder()
	srv.ExportStudiesCSV(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	body := rr.Body.String()
	lines := strings.Split(strings.TrimSpace(body), "\n")
	assert.Len(t, lines, 1, "expected only header row when label filter matches nothing")
}
