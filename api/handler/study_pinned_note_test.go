package handler_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
)

func TestStudyPinnedNoteCRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	// Create pinned note
	body := `{"body":"Important: check slice 42"}`
	req := httptest.NewRequest(http.MethodPost, "/api/studies/"+study.ID+"/pinned-notes", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", study.ID)
	w := httptest.NewRecorder()
	srv.CreateStudyPinnedNote(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "check slice 42")

	// List pinned notes
	req = httptest.NewRequest(http.MethodGet, "/api/studies/"+study.ID+"/pinned-notes", nil)
	req.SetPathValue("id", study.ID)
	w = httptest.NewRecorder()
	srv.ListStudyPinnedNotes(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "check slice 42")
}

func TestStudyPinnedNote_EmptyBody(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	body := `{"body":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/studies/"+study.ID+"/pinned-notes", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", study.ID)
	w := httptest.NewRecorder()
	srv.CreateStudyPinnedNote(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
