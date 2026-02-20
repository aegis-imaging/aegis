package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/msenjem/aegis/api/model"
	"github.com/msenjem/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateDestination_DicomwebRequiresURL(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body, _ := json.Marshal(map[string]any{
		"name": "Dest A",
		"type": "dicomweb",
	})
	req := httptest.NewRequest("POST", "/api/destinations", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	srv.CreateDestination(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestCreateDestination_DIMSERequiresFields(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body, _ := json.Marshal(map[string]any{
		"name":     "DIMSE A",
		"type":     "dimse",
		"ae_title": "REMOTE_AE",
		"port":     11112,
	})
	req := httptest.NewRequest("POST", "/api/destinations", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	srv.CreateDestination(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestCreateDestination_DIMSEValid(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body, _ := json.Marshal(map[string]any{
		"name":     "DIMSE A",
		"type":     "dimse",
		"ae_title": "REMOTE_AE",
		"host":     "10.0.0.5",
		"port":     11112,
	})
	req := httptest.NewRequest("POST", "/api/destinations", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	srv.CreateDestination(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)
	var d model.Destination
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&d))
	assert.Equal(t, "dimse", d.Type)
	assert.Equal(t, "REMOTE_AE", d.AETitle)
	assert.Equal(t, "10.0.0.5", d.Host)
	assert.Equal(t, 11112, d.Port)
}
