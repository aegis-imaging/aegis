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

func TestBulkReassignStudies(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study1 := testutil.CreateTestStudy(t, db, proj.ID)
	study2 := testutil.CreateTestStudy(t, db, proj.ID)

	// Create a second project to reassign to.
	var newProjectID string
	err := db.QueryRow(`INSERT INTO projects (name, slug) VALUES ('Target', 'target') RETURNING id`).Scan(&newProjectID)
	require.NoError(t, err)

	body := `{"study_ids":["` + study1.ID + `","` + study2.ID + `"],"new_project_id":"` + newProjectID + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/studies/bulk-reassign", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.BulkReassignStudies(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, float64(2), resp["processed"])
}

func TestBulkReassign_EmptyStudyIDs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body := `{"study_ids":[],"new_project_id":"00000000-0000-0000-0000-000000000001"}`
	req := httptest.NewRequest(http.MethodPost, "/api/studies/bulk-reassign", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.BulkReassignStudies(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestBulkReassign_MissingProjectID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body := `{"study_ids":["00000000-0000-0000-0000-000000000001"],"new_project_id":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/studies/bulk-reassign", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.BulkReassignStudies(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
