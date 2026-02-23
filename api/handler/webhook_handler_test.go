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

func TestListWebhooks_Empty(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest("GET", "/api/webhook-subscriptions", nil)
	rr := httptest.NewRecorder()
	srv.ListWebhooks(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var subs []model.WebhookSubscription
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&subs))
	assert.Empty(t, subs)
}

func TestCreateWebhook_OK(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body, _ := json.Marshal(map[string]any{
		"url":    "https://example.com/webhook",
		"events": []string{"study.approved"},
		"secret": "my-secret-key",
	})
	req := httptest.NewRequest("POST", "/api/webhook-subscriptions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.CreateWebhook(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)
	var sub model.WebhookSubscription
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&sub))
	assert.Equal(t, "https://example.com/webhook", sub.URL)
	assert.Equal(t, []string{"study.approved"}, sub.Events)
	assert.True(t, sub.Enabled)
	assert.NotEmpty(t, sub.ID)
}

func TestCreateWebhook_MissingURL(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body, _ := json.Marshal(map[string]any{
		"events": []string{"study.approved"},
		"secret": "key",
	})
	req := httptest.NewRequest("POST", "/api/webhook-subscriptions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.CreateWebhook(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestCreateWebhook_InvalidEvent(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body, _ := json.Marshal(map[string]any{
		"url":    "https://example.com/hook",
		"events": []string{"unknown.event"},
		"secret": "key",
	})
	req := httptest.NewRequest("POST", "/api/webhook-subscriptions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.CreateWebhook(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestGetWebhook_OK(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	// Create a webhook
	createBody, _ := json.Marshal(map[string]any{
		"url":    "https://example.com/webhook",
		"events": []string{"study.approved"},
		"secret": "key",
	})
	createReq := httptest.NewRequest("POST", "/api/webhook-subscriptions", bytes.NewBuffer(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createRR := httptest.NewRecorder()
	srv.CreateWebhook(createRR, createReq)
	require.Equal(t, http.StatusCreated, createRR.Code)

	var created model.WebhookSubscription
	require.NoError(t, json.NewDecoder(createRR.Body).Decode(&created))

	// Get it
	req := httptest.NewRequest("GET", "/api/webhook-subscriptions/"+created.ID, nil)
	req.SetPathValue("id", created.ID)
	rr := httptest.NewRecorder()
	srv.GetWebhook(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var sub model.WebhookSubscription
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&sub))
	assert.Equal(t, created.ID, sub.ID)
}

func TestGetWebhook_NotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest("GET", "/api/webhook-subscriptions/00000000-0000-0000-0000-000000000000", nil)
	req.SetPathValue("id", "00000000-0000-0000-0000-000000000000")
	rr := httptest.NewRecorder()
	srv.GetWebhook(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestDeleteWebhook_OK(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	// Create then delete
	createBody, _ := json.Marshal(map[string]any{
		"url":    "https://example.com/delete-me",
		"events": []string{"study.approved"},
		"secret": "key",
	})
	createReq := httptest.NewRequest("POST", "/api/webhook-subscriptions", bytes.NewBuffer(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createRR := httptest.NewRecorder()
	srv.CreateWebhook(createRR, createReq)
	require.Equal(t, http.StatusCreated, createRR.Code)

	var created model.WebhookSubscription
	require.NoError(t, json.NewDecoder(createRR.Body).Decode(&created))

	deleteReq := httptest.NewRequest("DELETE", "/api/webhook-subscriptions/"+created.ID, nil)
	deleteReq.SetPathValue("id", created.ID)
	deleteRR := httptest.NewRecorder()
	srv.DeleteWebhook(deleteRR, deleteReq)

	assert.Equal(t, http.StatusOK, deleteRR.Code)

	// Verify gone
	getReq := httptest.NewRequest("GET", "/api/webhook-subscriptions/"+created.ID, nil)
	getReq.SetPathValue("id", created.ID)
	getRR := httptest.NewRecorder()
	srv.GetWebhook(getRR, getReq)
	assert.Equal(t, http.StatusNotFound, getRR.Code)
}

func TestGetWebhookDeliveries_Empty(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	// Create a webhook subscription
	createBody, _ := json.Marshal(map[string]any{
		"url":    "https://example.com/hook",
		"events": []string{"study.approved"},
		"secret": "key",
	})
	createReq := httptest.NewRequest("POST", "/api/webhook-subscriptions", bytes.NewBuffer(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createRR := httptest.NewRecorder()
	srv.CreateWebhook(createRR, createReq)
	require.Equal(t, http.StatusCreated, createRR.Code)

	var created model.WebhookSubscription
	require.NoError(t, json.NewDecoder(createRR.Body).Decode(&created))

	// Get deliveries (should be empty)
	req := httptest.NewRequest("GET", "/api/webhook-subscriptions/"+created.ID+"/deliveries", nil)
	req.SetPathValue("id", created.ID)
	rr := httptest.NewRecorder()
	srv.GetWebhookDeliveries(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}
