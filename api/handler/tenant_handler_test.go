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

func TestCreateTenant_OK(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body, _ := json.Marshal(map[string]any{
		"slug": "acme",
		"name": "Acme Co",
	})
	req := httptest.NewRequest("POST", "/api/tenants", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.CreateTenant(rr, req)
	require.Equal(t, http.StatusCreated, rr.Code)

	var got model.Tenant
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&got))
	assert.NotEmpty(t, got.ID)
	assert.Equal(t, "acme", got.Slug)
	assert.Equal(t, "Acme Co", got.Name)
	assert.True(t, got.Enabled)
}

func TestCreateTenant_RejectsBadSlug(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body, _ := json.Marshal(map[string]any{
		"slug": "Has_Underscores", // uppercase + underscore both invalid
		"name": "Bad Slug Co",
	})
	req := httptest.NewRequest("POST", "/api/tenants", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.CreateTenant(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestCreateTenant_RejectsDuplicateSlug(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body, _ := json.Marshal(map[string]any{"slug": "dup-test", "name": "First"})
	req := httptest.NewRequest("POST", "/api/tenants", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.CreateTenant(rr, req)
	require.Equal(t, http.StatusCreated, rr.Code)

	// Second with same slug should fail.
	body2, _ := json.Marshal(map[string]any{"slug": "dup-test", "name": "Second"})
	req2 := httptest.NewRequest("POST", "/api/tenants", bytes.NewBuffer(body2))
	req2.Header.Set("Content-Type", "application/json")
	rr2 := httptest.NewRecorder()
	srv.CreateTenant(rr2, req2)
	assert.Equal(t, http.StatusBadRequest, rr2.Code)
}

func TestListTenants_ReturnsCreated(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	// Seed via the API itself.
	for _, slug := range []string{"alpha", "beta", "gamma"} {
		body, _ := json.Marshal(map[string]any{"slug": slug, "name": slug})
		req := httptest.NewRequest("POST", "/api/tenants", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		srv.CreateTenant(rr, req)
		require.Equal(t, http.StatusCreated, rr.Code)
	}

	listReq := httptest.NewRequest("GET", "/api/tenants", nil)
	listRR := httptest.NewRecorder()
	srv.ListTenants(listRR, listReq)
	require.Equal(t, http.StatusOK, listRR.Code)

	var listed []model.Tenant
	require.NoError(t, json.NewDecoder(listRR.Body).Decode(&listed))
	assert.GreaterOrEqual(t, len(listed), 3)
}

func TestUpdateTenant_RenamesAndTogglesEnabled(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	createBody, _ := json.Marshal(map[string]any{"slug": "upd-test", "name": "Original"})
	createReq := httptest.NewRequest("POST", "/api/tenants", bytes.NewBuffer(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createRR := httptest.NewRecorder()
	srv.CreateTenant(createRR, createReq)
	require.Equal(t, http.StatusCreated, createRR.Code)

	var created model.Tenant
	require.NoError(t, json.NewDecoder(createRR.Body).Decode(&created))

	disabled := false
	updateBody, _ := json.Marshal(map[string]any{
		"name":    "Renamed",
		"enabled": &disabled,
	})
	updateReq := httptest.NewRequest("PUT", "/api/tenants/"+created.ID, bytes.NewBuffer(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateReq.SetPathValue("id", created.ID)
	updateRR := httptest.NewRecorder()
	srv.UpdateTenant(updateRR, updateReq)
	require.Equal(t, http.StatusOK, updateRR.Code)

	var updated model.Tenant
	require.NoError(t, json.NewDecoder(updateRR.Body).Decode(&updated))
	assert.Equal(t, "Renamed", updated.Name)
	assert.False(t, updated.Enabled)
	assert.Equal(t, "upd-test", updated.Slug, "slug must be immutable post-create")
}

func TestGetTenant_NotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest("GET", "/api/tenants/00000000-0000-0000-0000-000000000000", nil)
	req.SetPathValue("id", "00000000-0000-0000-0000-000000000000")
	rr := httptest.NewRecorder()
	srv.GetTenant(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestDeleteTenant_OK(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	createBody, _ := json.Marshal(map[string]any{"slug": "del-test", "name": "Delete Me"})
	createReq := httptest.NewRequest("POST", "/api/tenants", bytes.NewBuffer(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createRR := httptest.NewRecorder()
	srv.CreateTenant(createRR, createReq)
	require.Equal(t, http.StatusCreated, createRR.Code)
	var created model.Tenant
	require.NoError(t, json.NewDecoder(createRR.Body).Decode(&created))

	deleteReq := httptest.NewRequest("DELETE", "/api/tenants/"+created.ID, nil)
	deleteReq.SetPathValue("id", created.ID)
	deleteRR := httptest.NewRecorder()
	srv.DeleteTenant(deleteRR, deleteReq)
	assert.Equal(t, http.StatusOK, deleteRR.Code)

	// Confirm gone.
	getReq := httptest.NewRequest("GET", "/api/tenants/"+created.ID, nil)
	getReq.SetPathValue("id", created.ID)
	getRR := httptest.NewRecorder()
	srv.GetTenant(getRR, getReq)
	assert.Equal(t, http.StatusNotFound, getRR.Code)
}
