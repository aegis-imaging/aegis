package handler_test

import (
	"bytes"
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

func TestRevokeShare_Handler(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	share, err := model.CreateExportShare(t.Context(), db, study.ID, "hash-revoke-ok",
		"recipient@test.com", "", "admin@test.com", time.Now().Add(24*time.Hour), nil)
	require.NoError(t, err)

	req := httptest.NewRequest("DELETE", "/api/shares/"+share.ID, nil)
	req.SetPathValue("shareID", share.ID)
	rr := httptest.NewRecorder()
	srv.RevokeShare(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var result map[string]string
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.Equal(t, "revoked", result["status"])
}

func TestRevokeShare_VerifiesRevokedAtSet(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	share, err := model.CreateExportShare(t.Context(), db, study.ID, "hash-revoke-verify",
		"user@test.com", "", "admin@test.com", time.Now().Add(48*time.Hour), nil)
	require.NoError(t, err)

	req := httptest.NewRequest("DELETE", "/api/shares/"+share.ID, nil)
	req.SetPathValue("shareID", share.ID)
	rr := httptest.NewRecorder()
	srv.RevokeShare(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	// Confirm revoked_at is now set via the list endpoint.
	listReq := httptest.NewRequest("GET", "/api/shares?status=revoked", nil)
	listRR := httptest.NewRecorder()
	srv.ListAllShares(listRR, listReq)

	require.Equal(t, http.StatusOK, listRR.Code)
	var resp allSharesResponse
	require.NoError(t, json.NewDecoder(listRR.Body).Decode(&resp))
	assert.Equal(t, 1, resp.Total)
}

func TestRevokeShare_WithReason(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	share, err := model.CreateExportShare(t.Context(), db, study.ID, "hash-revoke-reason",
		"user@test.com", "", "admin@test.com", time.Now().Add(24*time.Hour), nil)
	require.NoError(t, err)

	body := bytes.NewBufferString(`{"reason":"share sent to wrong recipient"}`)
	req := httptest.NewRequest("DELETE", "/api/shares/"+share.ID, body)
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("shareID", share.ID)
	rr := httptest.NewRecorder()
	srv.RevokeShare(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	// Confirm revocation_reason was stored.
	fetched, err := model.GetExportShareByID(t.Context(), db, share.ID)
	require.NoError(t, err)
	require.NotNil(t, fetched.RevocationReason)
	assert.Equal(t, "share sent to wrong recipient", *fetched.RevocationReason)
}

func TestRevokeShare_IdempotentSecondRevoke(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	share, err := model.CreateExportShare(t.Context(), db, study.ID, "hash-revoke-idem",
		"user@test.com", "", "admin@test.com", time.Now().Add(24*time.Hour), nil)
	require.NoError(t, err)

	// Revoke once.
	req1 := httptest.NewRequest("DELETE", "/api/shares/"+share.ID, nil)
	req1.SetPathValue("shareID", share.ID)
	rr1 := httptest.NewRecorder()
	srv.RevokeShare(rr1, req1)
	require.Equal(t, http.StatusOK, rr1.Code)

	// Revoke again — should still return 200 (UPDATE WHERE revoked_at IS NULL is a no-op).
	req2 := httptest.NewRequest("DELETE", "/api/shares/"+share.ID, nil)
	req2.SetPathValue("shareID", share.ID)
	rr2 := httptest.NewRecorder()
	srv.RevokeShare(rr2, req2)
	assert.Equal(t, http.StatusOK, rr2.Code)
}
