package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/msenjem/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthz_Healthy(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest("GET", "/healthz", nil)
	rr := httptest.NewRecorder()
	srv.Healthz(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var result map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.Equal(t, "ok", result["status"])
	assert.Equal(t, "healthy", result["database"])
	assert.Equal(t, "healthy", result["storage"])
}

func TestHealthz_SidecarsDisabled(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest("GET", "/healthz", nil)
	rr := httptest.NewRecorder()
	srv.Healthz(rr, req)

	var result map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	services := result["services"].(map[string]any)
	for name, status := range services {
		assert.Equal(t, "disabled", status, "sidecar %s should be disabled when URL not set", name)
	}
}
