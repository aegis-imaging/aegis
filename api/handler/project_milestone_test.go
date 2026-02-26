package handler_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
)

func TestProjectMilestoneCRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	// Create milestone
	body := `{"title":"100 studies received","description":"First 100 studies milestone"}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+proj.ID+"/milestones", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", proj.ID)
	w := httptest.NewRecorder()
	srv.CreateProjectMilestone(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "100 studies received")

	// List milestones
	req = httptest.NewRequest(http.MethodGet, "/api/projects/"+proj.ID+"/milestones", nil)
	req.SetPathValue("id", proj.ID)
	w = httptest.NewRecorder()
	srv.ListProjectMilestones(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "100 studies received")
}

func TestProjectMilestone_MissingTitle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	body := `{"description":"no title"}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+proj.ID+"/milestones", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", proj.ID)
	w := httptest.NewRecorder()
	srv.CreateProjectMilestone(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
