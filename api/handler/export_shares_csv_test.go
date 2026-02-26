package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExportSharesCSV_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/export-shares.csv", nil)
	w := httptest.NewRecorder()
	srv.ExportSharesCSV(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/csv", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Header().Get("Content-Disposition"), "attachment")
	assert.Contains(t, w.Header().Get("Content-Disposition"), ".csv")

	body := w.Body.String()
	// Header row must exist
	assert.Contains(t, body, "id,study_id,recipient_email")
}

func TestExportSharesCSV_WithShare(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	// Approve study and create a share
	_, err := db.Exec(`UPDATE studies SET status='approved' WHERE id=$1`, study.ID)
	require.NoError(t, err)
	_, err = db.Exec(`
		INSERT INTO export_shares (id, study_id, token_hash, recipient_email, expires_at, created_by)
		VALUES (gen_random_uuid(), $1, 'a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2', 'csv@example.com',
		        now() + interval '48 hours', 'test')`, study.ID)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/export-shares.csv", nil)
	w := httptest.NewRecorder()
	srv.ExportSharesCSV(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "csv@example.com")
}

func TestExportSharesCSV_StatusFilter(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	// ?status=active should return only active shares (none here)
	req := httptest.NewRequest(http.MethodGet, "/api/export-shares.csv?status=active", nil)
	w := httptest.NewRecorder()
	srv.ExportSharesCSV(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	// Just the header row
	assert.Contains(t, body, "id,study_id,recipient_email")
}
