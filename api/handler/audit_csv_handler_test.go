package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aegis-imaging/aegis/api/middleware"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
)

func withResearcherCSVAudit(r *http.Request, id, email string) *http.Request {
	ctx := context.WithValue(r.Context(), middleware.AuthUserContextKey(), &middleware.AuthUser{
		ID: id, Email: email, Role: "researcher",
	})
	return r.WithContext(ctx)
}

func TestExportAuditCSV_OK(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest("GET", "/api/audit.csv", nil)
	rr := httptest.NewRecorder()
	srv.ExportAuditCSV(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "text/csv", rr.Header().Get("Content-Type"))
	assert.Contains(t, rr.Header().Get("Content-Disposition"), "audit")
	// Verify CSV header row
	body := rr.Body.String()
	assert.True(t, strings.HasPrefix(body, "id,"), "CSV should start with id column header")
	assert.Contains(t, body, "action")
}

func TestExportAuditCSV_ActionFilter(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest("GET", "/api/audit.csv?action=study.approved", nil)
	rr := httptest.NewRecorder()
	srv.ExportAuditCSV(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "text/csv", rr.Header().Get("Content-Type"))
}

func TestExportAuditCSV_ResearcherRequiresProjectScope(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	researcher := testutil.CreateTestAdminUser(t, db, "audit-csv-researcher@test.com", "researcher")

	req := httptest.NewRequest("GET", "/api/audit.csv", nil)
	req = withResearcherCSVAudit(req, researcher.ID, researcher.Email)
	rr := httptest.NewRecorder()

	srv.ExportAuditCSV(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "project_id is required for researcher queries")
}
