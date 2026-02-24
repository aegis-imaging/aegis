package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetStorageQuota_SetAndClear(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	// Set a quota.
	body := `{"storage_quota_bytes": 1073741824}`
	req := httptest.NewRequest(http.MethodPut, "/api/projects/"+proj.ID+"/storage-quota", strings.NewReader(body))
	req.SetPathValue("id", proj.ID)
	rr := httptest.NewRecorder()
	srv.SetStorageQuota(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	var project map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&project))
	assert.Equal(t, float64(1073741824), project["storage_quota_bytes"])

	// Clear the quota (null).
	req2 := httptest.NewRequest(http.MethodPut, "/api/projects/"+proj.ID+"/storage-quota", strings.NewReader(`{"storage_quota_bytes": null}`))
	req2.SetPathValue("id", proj.ID)
	rr2 := httptest.NewRecorder()
	srv.SetStorageQuota(rr2, req2)
	assert.Equal(t, http.StatusOK, rr2.Code)

	var proj2 map[string]any
	require.NoError(t, json.NewDecoder(rr2.Body).Decode(&proj2))
	assert.Nil(t, proj2["storage_quota_bytes"])
}

func TestSetStorageQuota_InvalidValue(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	req := httptest.NewRequest(http.MethodPut, "/api/projects/"+proj.ID+"/storage-quota", strings.NewReader(`{"storage_quota_bytes": -1}`))
	req.SetPathValue("id", proj.ID)
	rr := httptest.NewRecorder()
	srv.SetStorageQuota(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestSetStorageQuota_ProjectNotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodPut, "/api/projects/nonexistent/storage-quota", strings.NewReader(`{"storage_quota_bytes": 1000}`))
	req.SetPathValue("id", "00000000-0000-0000-0000-000000000000")
	rr := httptest.NewRecorder()
	srv.SetStorageQuota(rr, req)
	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestGetStorageUsage_NoQuota(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/projects/"+proj.ID+"/storage-usage", nil)
	req.SetPathValue("id", proj.ID)
	rr := httptest.NewRecorder()
	srv.GetStorageUsage(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, float64(0), resp["used_bytes"])
	assert.Nil(t, resp["quota_bytes"])
	assert.Nil(t, resp["usage_pct"])
}

func TestGetStorageUsage_WithQuota(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	// Set a quota first.
	_, err := db.Exec(`UPDATE projects SET storage_quota_bytes = 5000000000 WHERE id = $1`, proj.ID)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/projects/"+proj.ID+"/storage-usage", nil)
	req.SetPathValue("id", proj.ID)
	rr := httptest.NewRecorder()
	srv.GetStorageUsage(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, float64(0), resp["used_bytes"])
	assert.Equal(t, float64(5000000000), resp["quota_bytes"])
	assert.Equal(t, float64(0), resp["usage_pct"])
}
