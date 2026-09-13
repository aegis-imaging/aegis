package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServeProjectBidsExport_NoBidsStudies(t *testing.T) {
	if testing.Short() {
		t.Log("integration test")
		t.Skip()
	}

	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	// No studies with bids_status='complete' → 404.
	req := httptest.NewRequest(http.MethodGet, "/api/projects/"+proj.ID+"/bids-export", nil)
	req.SetPathValue("id", proj.ID)
	w := httptest.NewRecorder()
	srv.ServeProjectBidsExport(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestServeProjectBidsExport_ProjectNotFound(t *testing.T) {
	if testing.Short() {
		t.Log("integration test")
		t.Skip()
	}

	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/projects/00000000-0000-0000-0000-000000000000/bids-export", nil)
	req.SetPathValue("id", "00000000-0000-0000-0000-000000000000")
	w := httptest.NewRecorder()
	srv.ServeProjectBidsExport(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestServeProjectBidsExport_WithBidsStudy(t *testing.T) {
	if testing.Short() {
		t.Log("integration test")
		t.Skip()
	}

	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	study := testutil.CreateTestStudy(t, db, proj.ID)

	// Mark study as approved + BIDS complete (no actual files on disk — handler skips missing dirs).
	_, err := db.Exec(`UPDATE studies SET status='approved', bids_status='complete', bids_required=true WHERE id=$1`, study.ID)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/projects/"+proj.ID+"/bids-export", nil)
	req.SetPathValue("id", proj.ID)
	w := httptest.NewRecorder()
	srv.ServeProjectBidsExport(w, req)

	// Handler returns 200 ZIP even when BIDS dirs don't exist on disk (they are silently skipped).
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/zip", w.Header().Get("Content-Type"))
	assert.Equal(t, "1", w.Header().Get("X-BIDS-Study-Count"))
	assert.NotEmpty(t, w.Header().Get("Content-Disposition"))
}

func TestServeProjectBidsExport_StatusFilter(t *testing.T) {
	if testing.Short() {
		t.Log("integration test")
		t.Skip()
	}

	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	study := testutil.CreateTestStudy(t, db, proj.ID)
	// Approved + BIDS complete.
	_, err := db.Exec(`UPDATE studies SET status='approved', bids_status='complete', bids_required=true WHERE id=$1`, study.ID)
	require.NoError(t, err)

	// Filter to "rejected" — should return 404 (no matching studies).
	req := httptest.NewRequest(http.MethodGet, "/api/projects/"+proj.ID+"/bids-export?status=rejected", nil)
	req.SetPathValue("id", proj.ID)
	w := httptest.NewRecorder()
	srv.ServeProjectBidsExport(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
