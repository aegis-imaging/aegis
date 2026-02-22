package handler_test

import (
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

type allSharesResponse struct {
	Shares []json.RawMessage `json:"shares"`
	Total  int               `json:"total"`
	Limit  int               `json:"limit"`
	Offset int               `json:"offset"`
}


func TestListAllShares_Empty(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest("GET", "/api/shares", nil)
	rr := httptest.NewRecorder()
	srv.ListAllShares(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var result allSharesResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.Equal(t, 0, result.Total)
	assert.Len(t, result.Shares, 0)
	assert.Equal(t, 50, result.Limit)
	assert.Equal(t, 0, result.Offset)
}

func TestListAllShares_ReturnsAll(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	expiresAt := time.Now().Add(24 * time.Hour)
	model.CreateExportShare(t.Context(), db, study.ID, "hash1", "alice@test.com", "", "admin@test.com", expiresAt)
	model.CreateExportShare(t.Context(), db, study.ID, "hash2", "bob@test.com", "", "admin@test.com", expiresAt)

	req := httptest.NewRequest("GET", "/api/shares", nil)
	rr := httptest.NewRecorder()
	srv.ListAllShares(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var result allSharesResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.Equal(t, 2, result.Total)
	assert.Len(t, result.Shares, 2)
}

func TestListAllShares_StatusActiveFilter(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	futureExpiry := time.Now().Add(24 * time.Hour)
	pastExpiry := time.Now().Add(-24 * time.Hour)

	model.CreateExportShare(t.Context(), db, study.ID, "hash-active", "alice@test.com", "", "admin", futureExpiry)
	model.CreateExportShare(t.Context(), db, study.ID, "hash-expired", "bob@test.com", "", "admin", pastExpiry)

	req := httptest.NewRequest("GET", "/api/shares?status=active", nil)
	rr := httptest.NewRecorder()
	srv.ListAllShares(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var result allSharesResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.Equal(t, 1, result.Total)
	assert.Len(t, result.Shares, 1)
}

func TestListAllShares_StatusExpiredFilter(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	futureExpiry := time.Now().Add(24 * time.Hour)
	pastExpiry := time.Now().Add(-24 * time.Hour)

	model.CreateExportShare(t.Context(), db, study.ID, "hash-active", "alice@test.com", "", "admin", futureExpiry)
	model.CreateExportShare(t.Context(), db, study.ID, "hash-expired", "bob@test.com", "", "admin", pastExpiry)

	req := httptest.NewRequest("GET", "/api/shares?status=expired", nil)
	rr := httptest.NewRecorder()
	srv.ListAllShares(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var result allSharesResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.Equal(t, 1, result.Total)
	assert.Len(t, result.Shares, 1)
}

func TestListAllShares_Pagination(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	expiresAt := time.Now().Add(24 * time.Hour)
	for i := 0; i < 5; i++ {
		model.CreateExportShare(t.Context(), db, study.ID,
			"hash-"+string(rune('a'+i)), "user@test.com", "", "admin", expiresAt)
	}

	req := httptest.NewRequest("GET", "/api/shares?limit=2&offset=0", nil)
	rr := httptest.NewRecorder()
	srv.ListAllShares(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var result allSharesResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.Equal(t, 5, result.Total)
	assert.Len(t, result.Shares, 2)
	assert.Equal(t, 2, result.Limit)
}
