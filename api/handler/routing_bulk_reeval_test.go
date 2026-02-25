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

func TestBulkReEvaluateRouting_NoStudies(t *testing.T) {
	if testing.Short() {
		t.Log("integration test")
		t.Skip()
	}

	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+proj.ID+"/re-evaluate-routing", nil)
	req.SetPathValue("id", proj.ID)
	w := httptest.NewRecorder()
	srv.BulkReEvaluateRouting(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Evaluated    int    `json:"evaluated"`
		StatusFilter string `json:"status_filter"`
		Errors       []any  `json:"errors"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.GreaterOrEqual(t, resp.Evaluated, 0)
	assert.Empty(t, resp.Errors)
}

func TestBulkReEvaluateRouting_WithStudies(t *testing.T) {
	if testing.Short() {
		t.Log("integration test")
		t.Skip()
	}

	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	// Create 3 studies in the project.
	s1 := testutil.CreateTestStudy(t, db, proj.ID)
	s2 := testutil.CreateTestStudy(t, db, proj.ID)
	s3 := testutil.CreateTestStudy(t, db, proj.ID)
	_ = s1; _ = s2; _ = s3

	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+proj.ID+"/re-evaluate-routing", nil)
	req.SetPathValue("id", proj.ID)
	w := httptest.NewRecorder()
	srv.BulkReEvaluateRouting(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Evaluated int   `json:"evaluated"`
		Errors    []any `json:"errors"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	// At least the 3 we created (seed data may add more).
	assert.GreaterOrEqual(t, resp.Evaluated, 3)
	assert.Empty(t, resp.Errors)

	// Audit entry should have been written.
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM audit_trail WHERE action = 'routing.bulk_reeval' AND resource_id = $1`, proj.ID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestBulkReEvaluateRouting_StatusFilter(t *testing.T) {
	if testing.Short() {
		t.Log("integration test")
		t.Skip()
	}

	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	// Create one study, then update it to "approved" status.
	s := testutil.CreateTestStudy(t, db, proj.ID)
	_, err := db.Exec(`UPDATE studies SET status = 'approved' WHERE id = $1`, s.ID)
	require.NoError(t, err)

	// Re-evaluate only "approved" studies.
	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+proj.ID+"/re-evaluate-routing?status=approved", nil)
	req.SetPathValue("id", proj.ID)
	w := httptest.NewRecorder()
	srv.BulkReEvaluateRouting(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Evaluated    int    `json:"evaluated"`
		StatusFilter string `json:"status_filter"`
		Errors       []any  `json:"errors"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "approved", resp.StatusFilter)
	assert.GreaterOrEqual(t, resp.Evaluated, 1)
	assert.Empty(t, resp.Errors)
}

func TestBulkReEvaluateRouting_NotFound(t *testing.T) {
	if testing.Short() {
		t.Log("integration test")
		t.Skip()
	}

	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodPost, "/api/projects/00000000-0000-0000-0000-000000000000/re-evaluate-routing", nil)
	req.SetPathValue("id", "00000000-0000-0000-0000-000000000000")
	w := httptest.NewRecorder()
	srv.BulkReEvaluateRouting(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
