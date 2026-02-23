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

func TestGetProjectPhiConfig_Defaults(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	req := httptest.NewRequest("GET", "/api/projects/"+proj.ID+"/phi-config", nil)
	req.SetPathValue("id", proj.ID)
	rr := httptest.NewRecorder()
	srv.GetProjectPhiConfig(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var cfg model.ProjectPhiConfig
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&cfg))
	assert.Equal(t, proj.ID, cfg.ProjectID)
	// Should return global defaults when no override set.
	assert.Equal(t, 0.4, cfg.ConfidenceThreshold)
	assert.Equal(t, 3, cfg.MinTextLength)
}

func TestUpdateProjectPhiConfig_OK(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	body, _ := json.Marshal(map[string]any{
		"confidence_threshold": 0.7,
		"min_text_length":      5,
	})
	req := httptest.NewRequest("PUT", "/api/projects/"+proj.ID+"/phi-config", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", proj.ID)
	rr := httptest.NewRecorder()
	srv.UpdateProjectPhiConfig(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var cfg model.ProjectPhiConfig
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&cfg))
	assert.Equal(t, proj.ID, cfg.ProjectID)
	assert.Equal(t, 0.7, cfg.ConfidenceThreshold)
	assert.Equal(t, 5, cfg.MinTextLength)
}

func TestUpdateProjectPhiConfig_PartialUpdate(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	// First set both values.
	firstBody, _ := json.Marshal(map[string]any{
		"confidence_threshold": 0.6,
		"min_text_length":      4,
	})
	req1 := httptest.NewRequest("PUT", "/api/projects/"+proj.ID+"/phi-config", bytes.NewBuffer(firstBody))
	req1.Header.Set("Content-Type", "application/json")
	req1.SetPathValue("id", proj.ID)
	rr1 := httptest.NewRecorder()
	srv.UpdateProjectPhiConfig(rr1, req1)
	require.Equal(t, http.StatusOK, rr1.Code)

	// Now only update the threshold, min_text_length should be preserved.
	partialBody, _ := json.Marshal(map[string]any{
		"confidence_threshold": 0.9,
	})
	req2 := httptest.NewRequest("PUT", "/api/projects/"+proj.ID+"/phi-config", bytes.NewBuffer(partialBody))
	req2.Header.Set("Content-Type", "application/json")
	req2.SetPathValue("id", proj.ID)
	rr2 := httptest.NewRecorder()
	srv.UpdateProjectPhiConfig(rr2, req2)

	assert.Equal(t, http.StatusOK, rr2.Code)
	var cfg model.ProjectPhiConfig
	require.NoError(t, json.NewDecoder(rr2.Body).Decode(&cfg))
	assert.Equal(t, 0.9, cfg.ConfidenceThreshold)
	assert.Equal(t, 4, cfg.MinTextLength) // preserved from first update
}

func TestUpdateProjectPhiConfig_InvalidThreshold(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	body, _ := json.Marshal(map[string]any{
		"confidence_threshold": 1.5, // > 1.0 is invalid
	})
	req := httptest.NewRequest("PUT", "/api/projects/"+proj.ID+"/phi-config", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", proj.ID)
	rr := httptest.NewRecorder()
	srv.UpdateProjectPhiConfig(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestUpdateProjectPhiConfig_InvalidMinTextLength(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	body, _ := json.Marshal(map[string]any{
		"min_text_length": 0, // < 1 is invalid
	})
	req := httptest.NewRequest("PUT", "/api/projects/"+proj.ID+"/phi-config", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", proj.ID)
	rr := httptest.NewRecorder()
	srv.UpdateProjectPhiConfig(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestGetProjectPhiConfig_ReflectsUpdate(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	// Update the config.
	updateBody, _ := json.Marshal(map[string]any{
		"confidence_threshold": 0.8,
		"min_text_length":      7,
	})
	updateReq := httptest.NewRequest("PUT", "/api/projects/"+proj.ID+"/phi-config", bytes.NewBuffer(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateReq.SetPathValue("id", proj.ID)
	updateRR := httptest.NewRecorder()
	srv.UpdateProjectPhiConfig(updateRR, updateReq)
	require.Equal(t, http.StatusOK, updateRR.Code)

	// GET should now return updated values.
	getReq := httptest.NewRequest("GET", "/api/projects/"+proj.ID+"/phi-config", nil)
	getReq.SetPathValue("id", proj.ID)
	getRR := httptest.NewRecorder()
	srv.GetProjectPhiConfig(getRR, getReq)

	assert.Equal(t, http.StatusOK, getRR.Code)
	var cfg model.ProjectPhiConfig
	require.NoError(t, json.NewDecoder(getRR.Body).Decode(&cfg))
	assert.Equal(t, 0.8, cfg.ConfidenceThreshold)
	assert.Equal(t, 7, cfg.MinTextLength)
}
