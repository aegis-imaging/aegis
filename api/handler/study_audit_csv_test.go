package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExportStudyAuditCSV_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/studies/00000000-0000-0000-0000-000000000000/audit.csv", nil)
	req.SetPathValue("id", "00000000-0000-0000-0000-000000000000")
	w := httptest.NewRecorder()
	srv.ExportStudyAuditCSV(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestExportStudyAuditCSV_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	req := httptest.NewRequest(http.MethodGet, "/api/studies/"+study.ID+"/audit.csv", nil)
	req.SetPathValue("id", study.ID)
	w := httptest.NewRecorder()
	srv.ExportStudyAuditCSV(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/csv", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Header().Get("Content-Disposition"), "attachment")
	assert.Contains(t, w.Header().Get("Content-Disposition"), ".csv")

	// Should have header row
	lines := strings.Split(strings.TrimSpace(w.Body.String()), "\n")
	require.GreaterOrEqual(t, len(lines), 1)
	assert.Contains(t, lines[0], "id,created_at,action,actor")
}

func TestExportStudyAuditCSV_WithEntries(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	// Create an audit entry for this study
	model.CreateAuditEntry(context.Background(), db, "study.approved", "test@example.com", "study", study.ID, "127.0.0.1", map[string]any{"reason": "looks good"})

	req := httptest.NewRequest(http.MethodGet, "/api/studies/"+study.ID+"/audit.csv", nil)
	req.SetPathValue("id", study.ID)
	w := httptest.NewRecorder()
	srv.ExportStudyAuditCSV(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	lines := strings.Split(strings.TrimSpace(w.Body.String()), "\n")
	require.GreaterOrEqual(t, len(lines), 2) // header + at least 1 data row
	assert.Contains(t, lines[1], "study.approved")
	assert.Contains(t, lines[1], "test@example.com")
}
