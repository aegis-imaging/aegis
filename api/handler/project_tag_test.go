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

func TestProjectTags_CRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	// Add tag
	body := `{"tag":"neuroimaging"}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+proj.ID+"/tags", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", proj.ID)
	w := httptest.NewRecorder()
	srv.AddProjectTag(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	var tag model.ProjectTag
	require.NoError(t, json.NewDecoder(w.Body).Decode(&tag))
	assert.Equal(t, "neuroimaging", tag.Tag)

	// List tags
	req = httptest.NewRequest(http.MethodGet, "/api/projects/"+proj.ID+"/tags", nil)
	req.SetPathValue("id", proj.ID)
	w = httptest.NewRecorder()
	srv.ListProjectTags(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var tags []model.ProjectTag
	require.NoError(t, json.NewDecoder(w.Body).Decode(&tags))
	assert.Len(t, tags, 1)

	// Delete tag
	req = httptest.NewRequest(http.MethodDelete, "/api/projects/"+proj.ID+"/tags/"+tag.ID, nil)
	req.SetPathValue("id", proj.ID)
	req.SetPathValue("tagID", tag.ID)
	w = httptest.NewRecorder()
	srv.DeleteProjectTag(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestProjectTag_EmptyTag(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	body := `{"tag":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+proj.ID+"/tags", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", proj.ID)
	w := httptest.NewRecorder()
	srv.AddProjectTag(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
