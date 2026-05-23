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

func TestCreateInstitutionEnrollmentToken_ReturnsTokenOnce(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	inst := testutil.CreateTestInstitution(t, db, "token-mint-once")

	req := httptest.NewRequest(http.MethodPost,
		"/api/institutions/"+inst.ID+"/enrollment-tokens",
		strings.NewReader(`{"label":"laptop-test","ttl_hours":24}`))
	req.SetPathValue("id", inst.ID)
	rr := httptest.NewRecorder()
	srv.CreateInstitutionEnrollmentToken(rr, req)
	require.Equal(t, http.StatusCreated, rr.Code, rr.Body.String())

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.NotEmpty(t, resp["token"], "raw token must be returned at creation time")
	assert.NotEmpty(t, resp["id"])
	assert.Contains(t, resp["install_command"], "--site-token=")
	assert.Contains(t, resp["install_command"], "--site-name=")
}

func TestListInstitutionEnrollmentTokens_OmitsRawToken(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	inst := testutil.CreateTestInstitution(t, db, "token-list")

	for i := 0; i < 2; i++ {
		create := httptest.NewRequest(http.MethodPost,
			"/api/institutions/"+inst.ID+"/enrollment-tokens",
			strings.NewReader(`{"label":"t"}`))
		create.SetPathValue("id", inst.ID)
		rr := httptest.NewRecorder()
		srv.CreateInstitutionEnrollmentToken(rr, create)
		require.Equal(t, http.StatusCreated, rr.Code)
	}

	list := httptest.NewRequest(http.MethodGet,
		"/api/institutions/"+inst.ID+"/enrollment-tokens", nil)
	list.SetPathValue("id", inst.ID)
	rr := httptest.NewRecorder()
	srv.ListInstitutionEnrollmentTokens(rr, list)
	require.Equal(t, http.StatusOK, rr.Code)

	var resp struct {
		Tokens []map[string]any `json:"tokens"`
		Count  int              `json:"count"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, 2, resp.Count)
	for _, tk := range resp.Tokens {
		_, hasToken := tk["token"]
		assert.False(t, hasToken, "list response must not echo the raw token field")
		assert.Equal(t, "active", tk["status"])
	}
}

func TestRevokeInstitutionEnrollmentToken(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	inst := testutil.CreateTestInstitution(t, db, "token-revoke")

	create := httptest.NewRequest(http.MethodPost,
		"/api/institutions/"+inst.ID+"/enrollment-tokens",
		strings.NewReader(`{"label":"to-revoke"}`))
	create.SetPathValue("id", inst.ID)
	createRR := httptest.NewRecorder()
	srv.CreateInstitutionEnrollmentToken(createRR, create)
	require.Equal(t, http.StatusCreated, createRR.Code)
	var created map[string]any
	require.NoError(t, json.NewDecoder(createRR.Body).Decode(&created))
	tokenID := created["id"].(string)

	revoke := httptest.NewRequest(http.MethodDelete,
		"/api/institutions/"+inst.ID+"/enrollment-tokens/"+tokenID, nil)
	revoke.SetPathValue("id", inst.ID)
	revoke.SetPathValue("tokenID", tokenID)
	revRR := httptest.NewRecorder()
	srv.RevokeInstitutionEnrollmentToken(revRR, revoke)
	assert.Equal(t, http.StatusOK, revRR.Code)

	// List should now show revoked status.
	list := httptest.NewRequest(http.MethodGet,
		"/api/institutions/"+inst.ID+"/enrollment-tokens", nil)
	list.SetPathValue("id", inst.ID)
	listRR := httptest.NewRecorder()
	srv.ListInstitutionEnrollmentTokens(listRR, list)
	var resp struct {
		Tokens []map[string]any `json:"tokens"`
	}
	require.NoError(t, json.NewDecoder(listRR.Body).Decode(&resp))
	require.Len(t, resp.Tokens, 1)
	assert.Equal(t, "revoked", resp.Tokens[0]["status"])
}
