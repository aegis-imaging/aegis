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

func TestListDigestSubscriptions_Empty(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/digest-subscriptions", nil)
	rr := httptest.NewRecorder()

	srv.ListDigestSubscriptions(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var result []model.DigestSubscription
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.Empty(t, result)
}

func TestCreateDigestSubscription_Weekly(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	body := map[string]any{
		"email":     "analyst@example.com",
		"frequency": "weekly",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+proj.ID+"/digest-subscriptions", bytes.NewReader(b))
	req.SetPathValue("projectID", proj.ID)
	rr := httptest.NewRecorder()

	srv.CreateDigestSubscription(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)
	var result model.DigestSubscription
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.NotEmpty(t, result.ID)
	assert.Equal(t, "analyst@example.com", result.Email)
	assert.Equal(t, "weekly", result.Frequency)
	assert.Equal(t, proj.ID, result.ProjectID)
	assert.True(t, result.Enabled)
}

func TestCreateDigestSubscription_Monthly(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	body := map[string]any{
		"email":     "monthly@example.com",
		"frequency": "monthly",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+proj.ID+"/digest-subscriptions", bytes.NewReader(b))
	req.SetPathValue("projectID", proj.ID)
	rr := httptest.NewRecorder()

	srv.CreateDigestSubscription(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)
	var result model.DigestSubscription
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.Equal(t, "monthly", result.Frequency)
}

func TestCreateDigestSubscription_MissingEmail(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	body := map[string]any{"frequency": "weekly"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+proj.ID+"/digest-subscriptions", bytes.NewReader(b))
	req.SetPathValue("projectID", proj.ID)
	rr := httptest.NewRecorder()

	srv.CreateDigestSubscription(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "email is required")
}

func TestCreateDigestSubscription_InvalidFrequency(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	body := map[string]any{
		"email":     "test@example.com",
		"frequency": "daily",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+proj.ID+"/digest-subscriptions", bytes.NewReader(b))
	req.SetPathValue("projectID", proj.ID)
	rr := httptest.NewRecorder()

	srv.CreateDigestSubscription(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "frequency must be weekly or monthly")
}

func TestCreateDigestSubscription_ProjectNotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body := map[string]any{"email": "x@example.com", "frequency": "weekly"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/projects/nonexistent/digest-subscriptions", bytes.NewReader(b))
	req.SetPathValue("projectID", "no-such-project")
	rr := httptest.NewRecorder()

	srv.CreateDigestSubscription(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Contains(t, rr.Body.String(), "project not found")
}

func TestListDigestSubscriptions_ByProject(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	// Create two subscriptions for the project
	for _, email := range []string{"a@example.com", "b@example.com"} {
		body := map[string]any{"email": email, "frequency": "weekly"}
		b, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/api/projects/"+proj.ID+"/digest-subscriptions", bytes.NewReader(b))
		req.SetPathValue("projectID", proj.ID)
		rr := httptest.NewRecorder()
		srv.CreateDigestSubscription(rr, req)
		require.Equal(t, http.StatusCreated, rr.Code)
	}

	// List with project filter
	req := httptest.NewRequest(http.MethodGet, "/api/projects/"+proj.ID+"/digest-subscriptions", nil)
	req.SetPathValue("projectID", proj.ID)
	rr := httptest.NewRecorder()
	srv.ListDigestSubscriptions(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var result []model.DigestSubscription
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.Len(t, result, 2)
}

func TestListDigestSubscriptions_AllReturnsAll(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	for _, email := range []string{"c@example.com", "d@example.com"} {
		body := map[string]any{"email": email, "frequency": "monthly"}
		b, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/api/projects/"+proj.ID+"/digest-subscriptions", bytes.NewReader(b))
		req.SetPathValue("projectID", proj.ID)
		rr := httptest.NewRecorder()
		srv.CreateDigestSubscription(rr, req)
		require.Equal(t, http.StatusCreated, rr.Code)
	}

	// List without project filter (global)
	req := httptest.NewRequest(http.MethodGet, "/api/digest-subscriptions", nil)
	rr := httptest.NewRecorder()
	srv.ListDigestSubscriptions(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var result []model.DigestSubscription
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.GreaterOrEqual(t, len(result), 2)
}

func TestDeleteDigestSubscription_Handler(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	// Create one subscription
	body := map[string]any{"email": "delete-me@example.com", "frequency": "weekly"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+proj.ID+"/digest-subscriptions", bytes.NewReader(b))
	req.SetPathValue("projectID", proj.ID)
	rr := httptest.NewRecorder()
	srv.CreateDigestSubscription(rr, req)
	require.Equal(t, http.StatusCreated, rr.Code)

	var created model.DigestSubscription
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&created))

	// Delete it
	req2 := httptest.NewRequest(http.MethodDelete, "/api/digest-subscriptions/"+created.ID, nil)
	req2.SetPathValue("id", created.ID)
	rr2 := httptest.NewRecorder()
	srv.DeleteDigestSubscription(rr2, req2)

	assert.Equal(t, http.StatusNoContent, rr2.Code)

	// Verify gone from list
	req3 := httptest.NewRequest(http.MethodGet, "/api/projects/"+proj.ID+"/digest-subscriptions", nil)
	req3.SetPathValue("projectID", proj.ID)
	rr3 := httptest.NewRecorder()
	srv.ListDigestSubscriptions(rr3, req3)
	var remaining []model.DigestSubscription
	require.NoError(t, json.NewDecoder(rr3.Body).Decode(&remaining))
	assert.Empty(t, remaining)
}

func TestDeleteDigestSubscription_NotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodDelete, "/api/digest-subscriptions/missing", nil)
	req.SetPathValue("id", "no-such-subscription")
	rr := httptest.NewRecorder()

	srv.DeleteDigestSubscription(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Contains(t, rr.Body.String(), "subscription not found")
}
