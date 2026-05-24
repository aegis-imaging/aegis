package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListSatellites_EmptyWhenNoneEnrolled(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/satellites", nil)
	rr := httptest.NewRecorder()
	srv.ListSatellites(rr, req)
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

	var resp struct {
		Satellites []map[string]any `json:"spokes"`
		Count  int              `json:"count"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, 0, resp.Count)
	assert.NotNil(t, resp.Satellites)
}

func TestListSatellites_IncludesEnrolledAndPending(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	// Enrolled: has a thumbprint set.
	enrolled := testutil.CreateTestInstitution(t, db, "satellite-enrolled")
	require.NoError(t, model.SetInstitutionClientCert(context.Background(), db, enrolled.ID,
		"abcd1234", "CN=satellite-enrolled"))

	// Pending: no cert yet, but has an active enrollment token.
	pending := testutil.CreateTestInstitution(t, db, "satellite-pending")
	_, hash, _ := model.GenerateEnrollmentToken()
	tok := &model.SatelliteEnrollmentToken{
		InstitutionID: pending.ID,
		ExpiresAt:     time.Now().UTC().Add(2 * time.Hour),
	}
	require.NoError(t, model.CreateSatelliteEnrollmentToken(context.Background(), db, tok, hash))

	// Not a satellite at all: regular institution, no cert, no token.
	_ = testutil.CreateTestInstitution(t, db, "satellite-irrelevant")

	req := httptest.NewRequest(http.MethodGet, "/api/satellites", nil)
	rr := httptest.NewRecorder()
	srv.ListSatellites(rr, req)
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

	var resp struct {
		Satellites []map[string]any `json:"spokes"`
		Count  int              `json:"count"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, 2, resp.Count, "expected enrolled + pending; irrelevant should be excluded")

	byName := map[string]map[string]any{}
	for _, sp := range resp.Satellites {
		byName[sp["institution_name"].(string)] = sp
	}
	require.Contains(t, byName, "satellite-enrolled")
	assert.Equal(t, "abcd1234", byName["satellite-enrolled"]["cert_thumbprint"])

	require.Contains(t, byName, "satellite-pending")
	assert.Equal(t, "", byName["satellite-pending"]["cert_thumbprint"])
	assert.EqualValues(t, 1, byName["satellite-pending"]["active_token_count"])
}
