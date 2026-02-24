package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCloneProject_HappyPath(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	src := testutil.SeedProject(t, db)

	body, _ := json.Marshal(map[string]string{"name": "Cloned Project", "slug": "cloned-project"})
	req := httptest.NewRequest("POST", "/api/projects/"+src.ID+"/clone", bytes.NewReader(body))
	req.SetPathValue("id", src.ID)
	rr := httptest.NewRecorder()
	srv.CloneProject(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)
	var p model.Project
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&p))
	assert.Equal(t, "Cloned Project", p.Name)
	assert.Equal(t, "cloned-project", p.Slug)
	assert.NotEqual(t, src.ID, p.ID, "cloned project should have a new ID")
}

func TestCloneProject_DefaultName(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	src := testutil.SeedProject(t, db)

	// Empty body — should use "Copy of <source name>" default
	req := httptest.NewRequest("POST", "/api/projects/"+src.ID+"/clone", bytes.NewReader([]byte(`{}`)))
	req.SetPathValue("id", src.ID)
	rr := httptest.NewRecorder()
	srv.CloneProject(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)
	var p model.Project
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&p))
	assert.Contains(t, p.Name, src.Name, "default clone name should contain source name")
}

func TestCloneProject_SlugConflict(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	src := testutil.SeedProject(t, db)

	// Try to clone with the same slug as the source — should conflict
	body, _ := json.Marshal(map[string]string{"name": "Same Slug", "slug": src.Slug})
	req := httptest.NewRequest("POST", "/api/projects/"+src.ID+"/clone", bytes.NewReader(body))
	req.SetPathValue("id", src.ID)
	rr := httptest.NewRecorder()
	srv.CloneProject(rr, req)

	assert.Equal(t, http.StatusConflict, rr.Code)
}

func TestCloneProject_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest("POST", "/api/projects/nonexistent/clone", bytes.NewReader([]byte(`{}`)))
	req.SetPathValue("id", "nonexistent")
	rr := httptest.NewRecorder()
	srv.CloneProject(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}
