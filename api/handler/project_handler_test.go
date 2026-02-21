package handler_test

import (
	"bytes"
	"encoding/json"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/msenjem/aegis/api/model"
	"github.com/msenjem/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListProjects_Handler(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest("GET", "/api/projects", nil)
	rr := httptest.NewRecorder()
	srv.ListProjects(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var projects []model.Project
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&projects))
	assert.GreaterOrEqual(t, len(projects), 1, "default project should exist")
}

func TestCreateProject_Handler(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body, _ := json.Marshal(map[string]string{"name": "New Project"})
	req := httptest.NewRequest("POST", "/api/projects", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	srv.CreateProject(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)
	var p model.Project
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&p))
	assert.Equal(t, "new-project", p.Slug, "slug should be auto-generated from name")
	assert.NotEmpty(t, p.ID)
}

func TestCreateProject_MissingName(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body, _ := json.Marshal(map[string]string{"description": "no name"})
	req := httptest.NewRequest("POST", "/api/projects", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	srv.CreateProject(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestGetProject_Handler(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	// Get the default project ID first
	projects, _ := model.ListProjects(context.Background(), db)
	projID := projects[0].ID

	req := httptest.NewRequest("GET", "/api/projects/"+projID, nil)
	req.SetPathValue("id", projID)
	rr := httptest.NewRecorder()
	srv.GetProject(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var p model.Project
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&p))
	assert.Equal(t, projID, p.ID)
}

func TestGetProject_NotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest("GET", "/api/projects/nonexistent", nil)
	req.SetPathValue("id", "nonexistent")
	rr := httptest.NewRecorder()
	srv.GetProject(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}
