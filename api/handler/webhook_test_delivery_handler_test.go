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

func TestWebhookTestDelivery_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest("POST", "/api/webhook-subscriptions/00000000-0000-0000-0000-000000000000/test", nil)
	req.SetPathValue("id", "00000000-0000-0000-0000-000000000000")
	rr := httptest.NewRecorder()
	srv.TestWebhookDelivery(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestWebhookTestDelivery_ConnectionFailed(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	// Create a webhook subscription pointing at an unreachable URL.
	body := `{"url":"http://127.0.0.1:1/webhook","events":["study.approved"],"secret":"test-secret","enabled":true}`
	createReq := httptest.NewRequest("POST", "/api/webhook-subscriptions", strings.NewReader(body))
	createReq.Header.Set("Content-Type", "application/json")
	createRR := httptest.NewRecorder()
	srv.CreateWebhook(createRR, createReq)
	require.Equal(t, http.StatusCreated, createRR.Code)

	var created map[string]any
	require.NoError(t, json.NewDecoder(createRR.Body).Decode(&created))
	subID, ok := created["id"].(string)
	require.True(t, ok, "expected subscription id in response")

	// Now trigger test delivery — the HTTP call will fail (connection refused).
	testReq := httptest.NewRequest("POST", "/api/webhook-subscriptions/"+subID+"/test", nil)
	testReq.SetPathValue("id", subID)
	testRR := httptest.NewRecorder()
	srv.TestWebhookDelivery(testRR, testReq)

	// Handler always returns 200 regardless of delivery outcome.
	assert.Equal(t, http.StatusOK, testRR.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(testRR.Body).Decode(&resp))
	// Delivery to port 1 should have failed.
	assert.Equal(t, false, resp["success"])
	assert.NotEmpty(t, resp["error"])
	assert.Equal(t, "http://127.0.0.1:1/webhook", resp["url"])
}
