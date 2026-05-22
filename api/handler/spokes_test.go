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

func TestListSpokes_EmptyWhenNoneEnrolled(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/spokes", nil)
	rr := httptest.NewRecorder()
	srv.ListSpokes(rr, req)
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

	var resp struct {
		Spokes []map[string]any `json:"spokes"`
		Count  int              `json:"count"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, 0, resp.Count)
	assert.NotNil(t, resp.Spokes)
}

func TestListSpokes_IncludesEnrolledAndPending(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	// Enrolled: has a thumbprint set.
	enrolled := testutil.CreateTestInstitution(t, db, "spoke-enrolled")
	require.NoError(t, model.SetInstitutionClientCert(context.Background(), db, enrolled.ID,
		"abcd1234", "CN=spoke-enrolled"))

	// Pending: no cert yet, but has an active enrollment token.
	pending := testutil.CreateTestInstitution(t, db, "spoke-pending")
	_, hash, _ := model.GenerateEnrollmentToken()
	tok := &model.SpokeEnrollmentToken{
		InstitutionID: pending.ID,
		ExpiresAt:     time.Now().UTC().Add(2 * time.Hour),
	}
	require.NoError(t, model.CreateSpokeEnrollmentToken(context.Background(), db, tok, hash))

	// Not a spoke at all: regular institution, no cert, no token.
	_ = testutil.CreateTestInstitution(t, db, "spoke-irrelevant")

	req := httptest.NewRequest(http.MethodGet, "/api/spokes", nil)
	rr := httptest.NewRecorder()
	srv.ListSpokes(rr, req)
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

	var resp struct {
		Spokes []map[string]any `json:"spokes"`
		Count  int              `json:"count"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, 2, resp.Count, "expected enrolled + pending; irrelevant should be excluded")

	byName := map[string]map[string]any{}
	for _, sp := range resp.Spokes {
		byName[sp["institution_name"].(string)] = sp
	}
	require.Contains(t, byName, "spoke-enrolled")
	assert.Equal(t, "abcd1234", byName["spoke-enrolled"]["cert_thumbprint"])

	require.Contains(t, byName, "spoke-pending")
	assert.Equal(t, "", byName["spoke-pending"]["cert_thumbprint"])
	assert.EqualValues(t, 1, byName["spoke-pending"]["active_token_count"])
}
