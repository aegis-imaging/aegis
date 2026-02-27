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

type allSharesResponse struct {
	Shares []json.RawMessage `json:"shares"`
	Total  int               `json:"total"`
	Limit  int               `json:"limit"`
	Offset int               `json:"offset"`
}

type shareDownloadsResponse struct {
	ShareID   string                 `json:"share_id"`
	Downloads []model.ExportDownload `json:"downloads"`
	Total     int                    `json:"total"`
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
	model.CreateExportShare(t.Context(), db, study.ID, "hash1", "alice@test.com", "", "admin@test.com", expiresAt, nil)
	model.CreateExportShare(t.Context(), db, study.ID, "hash2", "bob@test.com", "", "admin@test.com", expiresAt, nil)

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

	model.CreateExportShare(t.Context(), db, study.ID, "hash-active", "alice@test.com", "", "admin", futureExpiry, nil)
	model.CreateExportShare(t.Context(), db, study.ID, "hash-expired", "bob@test.com", "", "admin", pastExpiry, nil)

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

	model.CreateExportShare(t.Context(), db, study.ID, "hash-active", "alice@test.com", "", "admin", futureExpiry, nil)
	model.CreateExportShare(t.Context(), db, study.ID, "hash-expired", "bob@test.com", "", "admin", pastExpiry, nil)

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
			"hash-"+string(rune('a'+i)), "user@test.com", "", "admin", expiresAt, nil)
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

func TestGetShareDownloads_Empty(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	share, err := model.CreateExportShare(t.Context(), db, study.ID, "hash-dl-empty",
		"viewer@test.com", "", "admin@test.com", time.Now().Add(24*time.Hour), nil)
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/api/shares/"+share.ID+"/downloads", nil)
	req.SetPathValue("shareID", share.ID)
	rr := httptest.NewRecorder()
	srv.GetShareDownloads(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var result shareDownloadsResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.Equal(t, share.ID, result.ShareID)
	assert.Equal(t, 0, result.Total)
	assert.NotNil(t, result.Downloads)
	assert.Len(t, result.Downloads, 0)
}

func TestGetShareDownloads_ReturnsDownloads(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	share, err := model.CreateExportShare(t.Context(), db, study.ID, "hash-dl-multi",
		"viewer@test.com", "", "admin@test.com", time.Now().Add(24*time.Hour), nil)
	require.NoError(t, err)

	require.NoError(t, model.CreateExportDownload(context.Background(), db, share.ID, "10.0.0.1"))
	require.NoError(t, model.CreateExportDownload(context.Background(), db, share.ID, "10.0.0.2"))

	req := httptest.NewRequest("GET", "/api/shares/"+share.ID+"/downloads", nil)
	req.SetPathValue("shareID", share.ID)
	rr := httptest.NewRecorder()
	srv.GetShareDownloads(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var result shareDownloadsResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.Equal(t, share.ID, result.ShareID)
	assert.Equal(t, 2, result.Total)
	assert.Len(t, result.Downloads, 2)
	// downloads are newest-first; both client IPs should be present
	ips := []string{result.Downloads[0].ClientIP, result.Downloads[1].ClientIP}
	assert.ElementsMatch(t, []string{"10.0.0.1", "10.0.0.2"}, ips)
}

func TestListAllShares_DownloadCountReflectsDownloads(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	expiresAt := time.Now().Add(24 * time.Hour)
	share, err := model.CreateExportShare(t.Context(), db, study.ID, "hash-cnt",
		"user@test.com", "", "admin", expiresAt, nil)
	require.NoError(t, err)

	// Seed 3 downloads for this share.
	for i := 0; i < 3; i++ {
		require.NoError(t, model.CreateExportDownload(context.Background(), db, share.ID, "192.168.1.1"))
	}

	req := httptest.NewRequest("GET", "/api/shares", nil)
	rr := httptest.NewRecorder()
	srv.ListAllShares(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var resp allSharesResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	require.Len(t, resp.Shares, 1)

	// Decode the raw share to check download_count.
	var s model.ExportShare
	require.NoError(t, json.Unmarshal(resp.Shares[0], &s))
	assert.Equal(t, 3, s.DownloadCount)
}

func TestListAllShares_ResearcherRequiresProjectScope(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	researcher := testutil.CreateTestAdminUser(t, db, "shares-researcher@test.com", "researcher")

	req := httptest.NewRequest("GET", "/api/shares", nil)
	req = withResearcherUser(req, researcher.ID, researcher.Email)
	rr := httptest.NewRecorder()

	srv.ListAllShares(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "project_id is required for researcher queries")
}

func TestListAllShares_ResearcherWithProjectScopeAllowed(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)
	researcher := testutil.CreateTestAdminUser(t, db, "shares-allowed@test.com", "researcher")

	require.NoError(t, model.CreateProjectMember(context.Background(), db, &model.ProjectMember{
		ProjectID:   proj.ID,
		AdminUserID: researcher.ID,
		Role:        "coordinator",
	}))

	_, err := model.CreateExportShare(t.Context(), db, study.ID, "hash-scoped", "scoped@test.com", "", "admin", time.Now().Add(24*time.Hour), nil)
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/api/shares?project_id="+proj.ID, nil)
	req = withResearcherUser(req, researcher.ID, researcher.Email)
	rr := httptest.NewRecorder()

	srv.ListAllShares(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var result allSharesResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.Equal(t, 1, result.Total)
	assert.Len(t, result.Shares, 1)
}

func TestGetExportAnalytics_ResearcherRequiresProjectScope(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	researcher := testutil.CreateTestAdminUser(t, db, "analytics-researcher@test.com", "researcher")

	req := httptest.NewRequest("GET", "/api/export-analytics", nil)
	req = withResearcherUser(req, researcher.ID, researcher.Email)
	rr := httptest.NewRecorder()

	srv.GetExportAnalytics(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "project_id is required for researcher queries")
}
