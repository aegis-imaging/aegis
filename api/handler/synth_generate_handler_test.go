package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
)

// GenerateSyntheticStudy calls the synth-service which is not available in tests.
// The handler exits early with 503 when SynthServiceURL is unconfigured.

func TestGenerateSyntheticStudy_ServiceUnconfigured(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body, _ := json.Marshal(map[string]any{
		"project_slug": "default",
		"slices":       20,
	})
	req := httptest.NewRequest("POST", "/api/studies/generate-synthetic", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.GenerateSyntheticStudy(rr, req)

	// SynthServiceURL is not configured in test server → 503.
	assert.Equal(t, http.StatusServiceUnavailable, rr.Code)
}

func TestGenerateSyntheticStudy_InvalidJSON(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest("POST", "/api/studies/generate-synthetic",
		bytes.NewBufferString("{not valid json"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.GenerateSyntheticStudy(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestGenerateSyntheticStudy_DefaultsApplied(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	// Empty body → handler applies defaults before hitting service check.
	// SynthServiceURL == "" means 503, not a defaults-related error.
	body, _ := json.Marshal(map[string]any{})
	req := httptest.NewRequest("POST", "/api/studies/generate-synthetic", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.GenerateSyntheticStudy(rr, req)

	// Still 503 because service is unconfigured — but confirms JSON parsing succeeds.
	assert.Equal(t, http.StatusServiceUnavailable, rr.Code)
}
