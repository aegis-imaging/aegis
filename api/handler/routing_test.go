package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTestDestination_NotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodPost, "/api/destinations/nonexistent-uuid/test", nil)
	req.SetPathValue("id", "nonexistent-uuid")
	rr := httptest.NewRecorder()
	srv.TestDestination(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestTestDestination_DicomwebSuccess(t *testing.T) {
	// Spin up a fake DICOMweb server that returns 200.
	fakeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/studies", r.URL.Path)
		w.Header().Set("Content-Type", "application/dicom+json")
		w.WriteHeader(http.StatusOK)
	}))
	defer fakeServer.Close()

	db := testutil.TestDB(t)
	dest := &model.Destination{
		Name:        "test-dicomweb",
		Slug:        "test-dicomweb",
		Type:        "dicomweb",
		DicomwebURL: fakeServer.URL,
		Enabled:     true,
	}
	require.NoError(t, model.CreateDestination(t.Context(), db, dest))

	srv := testutil.TestServer(t, db)
	req := httptest.NewRequest(http.MethodPost, "/api/destinations/"+dest.ID+"/test", nil)
	req.SetPathValue("id", dest.ID)
	rr := httptest.NewRecorder()
	srv.TestDestination(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var result map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &result))
	assert.Equal(t, true, result["success"])
	assert.Equal(t, "dicomweb", result["type"])
	assert.Equal(t, dest.ID, result["destination_id"])
	assert.Equal(t, float64(200), result["status_code"])
	assert.Empty(t, result["error"])
}

func TestTestDestination_DicomwebFailure(t *testing.T) {
	// Fake server returns 401 Unauthorized.
	fakeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer fakeServer.Close()

	db := testutil.TestDB(t)
	dest := &model.Destination{
		Name:        "test-dicomweb-fail",
		Slug:        "test-dicomweb-fail",
		Type:        "dicomweb",
		DicomwebURL: fakeServer.URL,
		Enabled:     true,
	}
	require.NoError(t, model.CreateDestination(t.Context(), db, dest))

	srv := testutil.TestServer(t, db)
	req := httptest.NewRequest(http.MethodPost, "/api/destinations/"+dest.ID+"/test", nil)
	req.SetPathValue("id", dest.ID)
	rr := httptest.NewRecorder()
	srv.TestDestination(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var result map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &result))
	assert.Equal(t, false, result["success"])
	assert.Equal(t, float64(401), result["status_code"])
	assert.NotEmpty(t, result["error"])
}

func TestTestDestination_DicomwebAuthHeader(t *testing.T) {
	// Verify the auth header is forwarded to the DICOMweb server.
	var receivedAuth string
	fakeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer fakeServer.Close()

	db := testutil.TestDB(t)
	dest := &model.Destination{
		Name:               "test-dicomweb-auth",
		Slug:               "test-dicomweb-auth",
		Type:               "dicomweb",
		DicomwebURL:        fakeServer.URL,
		DicomwebAuthHeader: "Bearer secret-token",
		Enabled:            true,
	}
	require.NoError(t, model.CreateDestination(t.Context(), db, dest))

	srv := testutil.TestServer(t, db)
	req := httptest.NewRequest(http.MethodPost, "/api/destinations/"+dest.ID+"/test", nil)
	req.SetPathValue("id", dest.ID)
	rr := httptest.NewRecorder()
	srv.TestDestination(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "Bearer secret-token", receivedAuth)
}

func TestTestDestination_DimseNoDimseReceiver(t *testing.T) {
	// When DIMSE receiver URL is not configured, returns error in result (not HTTP error).
	db := testutil.TestDB(t)
	dest := &model.Destination{
		Name:    "test-dimse",
		Slug:    "test-dimse",
		Type:    "dimse",
		AETitle: "REMOTE",
		Host:    "10.0.0.1",
		Port:    104,
		Enabled: true,
	}
	require.NoError(t, model.CreateDestination(t.Context(), db, dest))

	srv := testutil.TestServer(t, db) // DimseReceiverURL is empty in test server.
	req := httptest.NewRequest(http.MethodPost, "/api/destinations/"+dest.ID+"/test", nil)
	req.SetPathValue("id", dest.ID)
	rr := httptest.NewRecorder()
	srv.TestDestination(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var result map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &result))
	assert.Equal(t, false, result["success"])
	assert.Contains(t, result["error"], "dimse receiver service not configured")
}
