package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStudyComments_CRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	// Create comment
	body := `{"body":"This is a test comment"}`
	req := httptest.NewRequest(http.MethodPost, "/api/studies/"+study.ID+"/comments", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", study.ID)
	w := httptest.NewRecorder()
	srv.CreateStudyComment(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	var comment model.StudyComment
	require.NoError(t, json.NewDecoder(w.Body).Decode(&comment))
	assert.Equal(t, "This is a test comment", comment.Body)

	// List comments
	req = httptest.NewRequest(http.MethodGet, "/api/studies/"+study.ID+"/comments", nil)
	req.SetPathValue("id", study.ID)
	w = httptest.NewRecorder()
	srv.ListStudyComments(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var comments []model.StudyComment
	require.NoError(t, json.NewDecoder(w.Body).Decode(&comments))
	assert.Len(t, comments, 1)

	// Delete comment
	req = httptest.NewRequest(http.MethodDelete, "/api/studies/"+study.ID+"/comments/"+comment.ID, nil)
	req.SetPathValue("id", study.ID)
	req.SetPathValue("commentID", comment.ID)
	w = httptest.NewRecorder()
	srv.DeleteStudyComment(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestStudyComment_EmptyBody(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	body := `{"body":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/studies/"+study.ID+"/comments", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", study.ID)
	w := httptest.NewRecorder()
	srv.CreateStudyComment(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestStudyComment_StudyNotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body := `{"body":"test"}`
	req := httptest.NewRequest(http.MethodPost, "/api/studies/00000000-0000-0000-0000-000000000099/comments", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "00000000-0000-0000-0000-000000000099")
	w := httptest.NewRecorder()
	srv.CreateStudyComment(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}
