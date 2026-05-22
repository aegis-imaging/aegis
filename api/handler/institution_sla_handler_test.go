package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetInstitutionSLA_OK(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	inst := testutil.CreateTestInstitution(t, db, "sla-test-inst")

	req := httptest.NewRequest(http.MethodGet, "/api/institutions/"+inst.ID+"/sla", nil)
	req.SetPathValue("id", inst.ID)
	rr := httptest.NewRecorder()
	srv.GetInstitutionSLA(rr, req)
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, float64(30), resp["days"], "default window should be 30 days")

	inst2, ok := resp["institution"].(map[string]any)
	require.True(t, ok, "institution payload missing")
	assert.Equal(t, "sla-test-inst", inst2["institution_name"])
	assert.Equal(t, float64(0), inst2["total_studies"])
}

func TestGetInstitutionSLA_CustomDays(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	inst := testutil.CreateTestInstitution(t, db, "sla-custom-days")

	req := httptest.NewRequest(http.MethodGet, "/api/institutions/"+inst.ID+"/sla?days=7", nil)
	req.SetPathValue("id", inst.ID)
	rr := httptest.NewRecorder()
	srv.GetInstitutionSLA(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, float64(7), resp["days"])
}

func TestGetInstitutionSLA_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/institutions/00000000-0000-0000-0000-000000000000/sla", nil)
	req.SetPathValue("id", "00000000-0000-0000-0000-000000000000")
	rr := httptest.NewRecorder()
	srv.GetInstitutionSLA(rr, req)
	assert.Equal(t, http.StatusNotFound, rr.Code)
}
