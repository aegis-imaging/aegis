package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetStuckStudies_Empty(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest("GET", "/api/studies/stuck", nil)
	rr := httptest.NewRecorder()
	srv.GetStuckStudies(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp struct {
		Stuck   []model.Study `json:"stuck"`
		Total   int           `json:"total"`
		Minutes int           `json:"minutes"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Empty(t, resp.Stuck)
	assert.Equal(t, 0, resp.Total)
	assert.Equal(t, 60, resp.Minutes, "default threshold should be 60 minutes")
}

func TestGetStuckStudies_DefaultThreshold(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	testutil.CreateTestStudy(t, db, proj.ID)

	// Using default 60 minutes — a freshly created study is not stuck yet
	req := httptest.NewRequest("GET", "/api/studies/stuck", nil)
	rr := httptest.NewRecorder()
	srv.GetStuckStudies(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp struct {
		Stuck   []model.Study `json:"stuck"`
		Total   int           `json:"total"`
		Minutes int           `json:"minutes"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, 0, resp.Total, "fresh study should not be stuck")
}

func TestGetStuckStudies_ZeroMinutesThreshold(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	testutil.CreateTestStudy(t, db, proj.ID)

	// Using 0 minutes (all non-terminal studies are "stuck")
	req := httptest.NewRequest("GET", "/api/studies/stuck?minutes=0", nil)
	rr := httptest.NewRecorder()
	srv.GetStuckStudies(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp struct {
		Stuck   []model.Study `json:"stuck"`
		Total   int           `json:"total"`
		Minutes int           `json:"minutes"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	// minutes=0 is invalid (< 1), so handler should use default 60
	assert.Equal(t, 60, resp.Minutes, "invalid 0 minutes should default to 60")
}

func TestGetStuckStudies_ProjectFilter(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	testutil.CreateTestStudy(t, db, proj.ID)

	req := httptest.NewRequest("GET", "/api/studies/stuck?project_id="+proj.ID+"&minutes=1", nil)
	rr := httptest.NewRecorder()
	srv.GetStuckStudies(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp struct {
		Total   int `json:"total"`
		Minutes int `json:"minutes"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, 1, resp.Minutes)
}
