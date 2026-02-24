package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetProjectSLAThreshold_Set(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	proj := testutil.SeedProject(t, db)

	body, _ := json.Marshal(map[string]int{"stuck_threshold_minutes": 120})
	req := httptest.NewRequest("PUT", "/api/projects/"+proj.ID+"/sla-threshold", bytes.NewReader(body))
	req.SetPathValue("id", proj.ID)
	rr := httptest.NewRecorder()
	srv.SetProjectSLAThreshold(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var p model.Project
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&p))
	require.NotNil(t, p.StuckThresholdMinutes)
	assert.Equal(t, 120, *p.StuckThresholdMinutes)
}

func TestSetProjectSLAThreshold_Clear(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	proj := testutil.SeedProject(t, db)

	// First set a threshold
	mins := 90
	require.NoError(t, model.UpdateProjectSLAThreshold(context.Background(), db, proj.ID, &mins))

	// Now clear it with null
	body := []byte(`{"stuck_threshold_minutes": null}`)
	req := httptest.NewRequest("PUT", "/api/projects/"+proj.ID+"/sla-threshold", bytes.NewReader(body))
	req.SetPathValue("id", proj.ID)
	rr := httptest.NewRecorder()
	srv.SetProjectSLAThreshold(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var p model.Project
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&p))
	assert.Nil(t, p.StuckThresholdMinutes)
}

func TestSetProjectSLAThreshold_InvalidZero(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	proj := testutil.SeedProject(t, db)

	body, _ := json.Marshal(map[string]int{"stuck_threshold_minutes": 0})
	req := httptest.NewRequest("PUT", "/api/projects/"+proj.ID+"/sla-threshold", bytes.NewReader(body))
	req.SetPathValue("id", proj.ID)
	rr := httptest.NewRecorder()
	srv.SetProjectSLAThreshold(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestSetProjectSLAThreshold_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body, _ := json.Marshal(map[string]int{"stuck_threshold_minutes": 30})
	req := httptest.NewRequest("PUT", "/api/projects/nonexistent/sla-threshold", bytes.NewReader(body))
	req.SetPathValue("id", "nonexistent")
	rr := httptest.NewRecorder()
	srv.SetProjectSLAThreshold(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

// TestGetStuckStudies_PerProjectThreshold verifies that when a project has
// stuck_threshold_minutes set, it is used as the default when project_id is scoped.
func TestGetStuckStudies_PerProjectThreshold(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	proj := testutil.SeedProject(t, db)

	// Set per-project threshold to 120 minutes
	mins := 120
	require.NoError(t, model.UpdateProjectSLAThreshold(context.Background(), db, proj.ID, &mins))

	// Without explicit ?minutes= the per-project threshold should be used
	req := httptest.NewRequest("GET", "/api/studies/stuck?project_id="+proj.ID, nil)
	rr := httptest.NewRecorder()
	srv.GetStuckStudies(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp struct {
		Minutes int `json:"minutes"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, 120, resp.Minutes, "should use per-project threshold of 120 minutes")
}

// TestGetStuckStudies_ExplicitMinutesOverridesProject verifies that ?minutes=
// still overrides the per-project threshold.
func TestGetStuckStudies_ExplicitMinutesOverridesProject(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	proj := testutil.SeedProject(t, db)

	// Set per-project threshold to 120 minutes
	mins := 120
	require.NoError(t, model.UpdateProjectSLAThreshold(context.Background(), db, proj.ID, &mins))

	// Explicit ?minutes=30 should override the project threshold
	req := httptest.NewRequest("GET", "/api/studies/stuck?project_id="+proj.ID+"&minutes=30", nil)
	rr := httptest.NewRecorder()
	srv.GetStuckStudies(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp struct {
		Minutes int `json:"minutes"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, 30, resp.Minutes, "explicit ?minutes= should override per-project threshold")
}
