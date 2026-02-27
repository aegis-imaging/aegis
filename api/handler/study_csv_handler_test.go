package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aegis-imaging/aegis/api/middleware"
	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withResearcherCSV(r *http.Request, id, email string) *http.Request {
	ctx := context.WithValue(r.Context(), middleware.AuthUserContextKey(), &middleware.AuthUser{
		ID: id, Email: email, Role: "researcher",
	})
	return r.WithContext(ctx)
}

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

func TestExportStudiesCSV_ResearcherRequiresProjectScope(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	testutil.CreateTestStudy(t, db, proj.ID)
	researcher := testutil.CreateTestAdminUser(t, db, "csv-researcher@test.com", "researcher")

	req := httptest.NewRequest("GET", "/api/studies.csv", nil)
	req = withResearcherCSV(req, researcher.ID, researcher.Email)
	rr := httptest.NewRecorder()
	srv.ExportStudiesCSV(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "project_id is required for researcher queries")
}

func TestExportStudiesCSV_ResearcherWithProjectScopeAllowed(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	testutil.CreateTestStudy(t, db, proj.ID)
	researcher := testutil.CreateTestAdminUser(t, db, "csv-researcher-allowed@test.com", "researcher")

	err := model.CreateProjectMember(context.Background(), db, &model.ProjectMember{
		ProjectID:   proj.ID,
		AdminUserID: researcher.ID,
		Role:        "coordinator",
	})
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/api/studies.csv?project_id="+proj.ID, nil)
	req = withResearcherCSV(req, researcher.ID, researcher.Email)
	rr := httptest.NewRecorder()
	srv.ExportStudiesCSV(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	body := rr.Body.String()
	lines := strings.Split(strings.TrimSpace(body), "\n")
	assert.GreaterOrEqual(t, len(lines), 2, "expected header + at least one data row")
}
