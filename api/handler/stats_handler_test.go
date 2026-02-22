package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type statsResp struct {
	StudyCounts struct {
		Received int `json:"received"`
		Defacing int `json:"defacing"`
		Clean    int `json:"clean"`
		Defaced  int `json:"defaced"`
		Approved int `json:"approved"`
		Rejected int `json:"rejected"`
		Total    int `json:"total"`
	} `json:"study_counts"`
	ActiveShares int    `json:"active_shares"`
	GeneratedAt  string `json:"generated_at"`
}

func TestGetStats_Empty(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest("GET", "/api/stats", nil)
	rr := httptest.NewRecorder()
	srv.GetStats(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var result statsResp
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.Equal(t, 0, result.StudyCounts.Total)
	assert.Equal(t, 0, result.ActiveShares)
	assert.NotEmpty(t, result.GeneratedAt)
}

func TestGetStats_CountsByStatus(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	// Create studies in different statuses.
	s1 := testutil.CreateTestStudy(t, db, proj.ID) // default status = received
	s2 := testutil.CreateTestStudy(t, db, proj.ID)
	s3 := testutil.CreateTestStudy(t, db, proj.ID)

	require.NoError(t, model.UpdateStudyStatus(context.Background(), db, s2.ID, "approved"))
	require.NoError(t, model.UpdateStudyStatus(context.Background(), db, s3.ID, "rejected"))
	_ = s1

	req := httptest.NewRequest("GET", "/api/stats", nil)
	rr := httptest.NewRecorder()
	srv.GetStats(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var result statsResp
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.Equal(t, 3, result.StudyCounts.Total)
	assert.Equal(t, 1, result.StudyCounts.Received)
	assert.Equal(t, 1, result.StudyCounts.Approved)
	assert.Equal(t, 1, result.StudyCounts.Rejected)
	assert.Equal(t, 0, result.StudyCounts.Defacing)
}

func TestGetStats_ActiveShareCount(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	future := time.Now().Add(24 * time.Hour)
	past := time.Now().Add(-24 * time.Hour)

	model.CreateExportShare(t.Context(), db, study.ID, "h1", "a@test.com", "", "admin", future, nil) // active
	model.CreateExportShare(t.Context(), db, study.ID, "h2", "b@test.com", "", "admin", past, nil)   // expired
	model.CreateExportShare(t.Context(), db, study.ID, "h3", "c@test.com", "", "admin", future, nil) // active

	req := httptest.NewRequest("GET", "/api/stats", nil)
	rr := httptest.NewRecorder()
	srv.GetStats(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var result statsResp
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.Equal(t, 2, result.ActiveShares)
}
