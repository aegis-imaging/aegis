package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListAPIKeys_Empty(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest("GET", "/api/api-keys", nil)
	rr := httptest.NewRecorder()
	srv.ListAPIKeys(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var keys []model.APIKey
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&keys))
	assert.Empty(t, keys)
}

func TestCreateAPIKey_OK(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body := `{"name":"Test Key"}`
	req := httptest.NewRequest("POST", "/api/api-keys", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.CreateAPIKey(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)
	var resp struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Key    string `json:"key"`
		Prefix string `json:"prefix"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, "Test Key", resp.Name)
	assert.True(t, strings.HasPrefix(resp.Key, "aegis_"), "key should start with aegis_")
	assert.NotEmpty(t, resp.ID)
}

func TestCreateAPIKey_MissingName(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body := `{"name":""}`
	req := httptest.NewRequest("POST", "/api/api-keys", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.CreateAPIKey(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestCreateAPIKey_InvalidExpiry(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	expiry := "2020-01-01T00:00:00Z" // past
	body, _ := json.Marshal(map[string]any{
		"name":       "Expired Key",
		"expires_at": expiry,
	})
	req := httptest.NewRequest("POST", "/api/api-keys", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.CreateAPIKey(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestEnableDisableAPIKey_OK(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	// Create a key first
	createBody := `{"name":"Key to toggle"}`
	createReq := httptest.NewRequest("POST", "/api/api-keys", bytes.NewBufferString(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createRR := httptest.NewRecorder()
	srv.CreateAPIKey(createRR, createReq)
	require.Equal(t, http.StatusCreated, createRR.Code)

	var created struct{ ID string `json:"id"` }
	require.NoError(t, json.NewDecoder(createRR.Body).Decode(&created))

	// Disable
	disableReq := httptest.NewRequest("PATCH", "/api/api-keys/"+created.ID+"/disable", nil)
	disableReq.SetPathValue("id", created.ID)
	disableRR := httptest.NewRecorder()
	srv.DisableAPIKey(disableRR, disableReq)
	assert.Equal(t, http.StatusOK, disableRR.Code)

	var disableResp struct{ Enabled bool `json:"enabled"` }
	require.NoError(t, json.NewDecoder(disableRR.Body).Decode(&disableResp))
	assert.False(t, disableResp.Enabled)

	// Re-enable
	enableReq := httptest.NewRequest("PATCH", "/api/api-keys/"+created.ID+"/enable", nil)
	enableReq.SetPathValue("id", created.ID)
	enableRR := httptest.NewRecorder()
	srv.EnableAPIKey(enableRR, enableReq)
	assert.Equal(t, http.StatusOK, enableRR.Code)

	var enableResp struct{ Enabled bool `json:"enabled"` }
	require.NoError(t, json.NewDecoder(enableRR.Body).Decode(&enableResp))
	assert.True(t, enableResp.Enabled)
}

func TestRotateAPIKey_OK(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	// Create a key first
	createReq := httptest.NewRequest("POST", "/api/api-keys", bytes.NewBufferString(`{"name":"Key to rotate"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createRR := httptest.NewRecorder()
	srv.CreateAPIKey(createRR, createReq)
	require.Equal(t, http.StatusCreated, createRR.Code)

	var created struct {
		ID  string `json:"id"`
		Key string `json:"key"`
	}
	require.NoError(t, json.NewDecoder(createRR.Body).Decode(&created))
	originalKey := created.Key

	// Rotate
	rotateReq := httptest.NewRequest("POST", "/api/api-keys/"+created.ID+"/rotate", nil)
	rotateReq.SetPathValue("id", created.ID)
	rotateRR := httptest.NewRecorder()
	srv.RotateAPIKey(rotateRR, rotateReq)
	require.Equal(t, http.StatusOK, rotateRR.Code)

	var rotated struct {
		ID  string `json:"id"`
		Key string `json:"key"`
	}
	require.NoError(t, json.NewDecoder(rotateRR.Body).Decode(&rotated))
	assert.Equal(t, created.ID, rotated.ID)
	assert.True(t, strings.HasPrefix(rotated.Key, "aegis_"), "rotated key should start with aegis_")
	assert.NotEqual(t, originalKey, rotated.Key, "rotated key must differ from original")
}

func TestRotateAPIKey_NotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest("POST", "/api/api-keys/nonexistent-id/rotate", nil)
	req.SetPathValue("id", "nonexistent-id")
	rr := httptest.NewRecorder()
	srv.RotateAPIKey(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestDeleteAPIKey_OK(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	// Create key
	createReq := httptest.NewRequest("POST", "/api/api-keys", bytes.NewBufferString(`{"name":"Key to delete"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createRR := httptest.NewRecorder()
	srv.CreateAPIKey(createRR, createReq)
	require.Equal(t, http.StatusCreated, createRR.Code)

	var created struct{ ID string `json:"id"` }
	require.NoError(t, json.NewDecoder(createRR.Body).Decode(&created))

	// Delete
	deleteReq := httptest.NewRequest("DELETE", "/api/api-keys/"+created.ID, nil)
	deleteReq.SetPathValue("id", created.ID)
	deleteRR := httptest.NewRecorder()
	srv.DeleteAPIKey(deleteRR, deleteReq)
	assert.Equal(t, http.StatusOK, deleteRR.Code)

	// Verify gone from list
	listReq := httptest.NewRequest("GET", "/api/api-keys", nil)
	listRR := httptest.NewRecorder()
	srv.ListAPIKeys(listRR, listReq)
	var keys []model.APIKey
	require.NoError(t, json.NewDecoder(listRR.Body).Decode(&keys))
	assert.Empty(t, keys)
}
