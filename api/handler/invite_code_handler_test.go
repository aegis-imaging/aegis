package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// POST /api/invite/validate
// ---------------------------------------------------------------------------

func TestValidateInviteCode_InvalidJSON(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest("POST", "/api/invite/validate", bytes.NewBufferString("not-json"))
	rr := httptest.NewRecorder()
	srv.ValidateInviteCode(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, false, resp["valid"])
}

func TestValidateInviteCode_EmptyCode(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body, _ := json.Marshal(map[string]string{"code": ""})
	req := httptest.NewRequest("POST", "/api/invite/validate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.ValidateInviteCode(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, false, resp["valid"])
}

func TestValidateInviteCode_UnknownCode(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body, _ := json.Marshal(map[string]string{"code": "DOES-NOT-EXIST"})
	req := httptest.NewRequest("POST", "/api/invite/validate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.ValidateInviteCode(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, false, resp["valid"])
}

func TestValidateInviteCode_ValidCode(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	ic, err := model.CreateInviteCode(context.Background(), db, "test-user")
	require.NoError(t, err)

	body, _ := json.Marshal(map[string]string{"code": ic.Code})
	req := httptest.NewRequest("POST", "/api/invite/validate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.ValidateInviteCode(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, true, resp["valid"])
}

func TestValidateInviteCode_RevokedCode(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	ic, err := model.CreateInviteCode(context.Background(), db, "revoked-user")
	require.NoError(t, err)
	require.NoError(t, model.RevokeInviteCode(context.Background(), db, ic.ID))

	body, _ := json.Marshal(map[string]string{"code": ic.Code})
	req := httptest.NewRequest("POST", "/api/invite/validate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.ValidateInviteCode(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, false, resp["valid"])
}

// ---------------------------------------------------------------------------
// POST /api/invite-codes  (create)
// ---------------------------------------------------------------------------

func TestCreateInviteCode_OK(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body, _ := json.Marshal(map[string]string{"label": "Dr. Smith"})
	req := httptest.NewRequest("POST", "/api/invite-codes", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.CreateInviteCode(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.NotEmpty(t, resp["id"])
	assert.NotEmpty(t, resp["code"])
	assert.Equal(t, "Dr. Smith", resp["label"])
	assert.Equal(t, true, resp["enabled"])
}

func TestCreateInviteCode_EmptyLabel(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	// Empty label is allowed (optional field).
	body, _ := json.Marshal(map[string]string{"label": ""})
	req := httptest.NewRequest("POST", "/api/invite-codes", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.CreateInviteCode(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)
}

// ---------------------------------------------------------------------------
// GET /api/invite-codes  (list)
// ---------------------------------------------------------------------------

func TestListInviteCodes_Empty(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest("GET", "/api/invite-codes", nil)
	rr := httptest.NewRecorder()
	srv.ListInviteCodes(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, float64(0), resp["total"])
}

func TestListInviteCodes_ReturnsCodes(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	_, err := model.CreateInviteCode(context.Background(), db, "alice")
	require.NoError(t, err)
	_, err = model.CreateInviteCode(context.Background(), db, "bob")
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/api/invite-codes", nil)
	rr := httptest.NewRecorder()
	srv.ListInviteCodes(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, float64(2), resp["total"])
}

// ---------------------------------------------------------------------------
// POST /api/invite-codes/{id}/revoke
// ---------------------------------------------------------------------------

func TestRevokeInviteCode_OK(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	ic, err := model.CreateInviteCode(context.Background(), db, "to-revoke")
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/api/invite-codes/"+ic.ID+"/revoke", nil)
	req.SetPathValue("id", ic.ID)
	rr := httptest.NewRecorder()
	srv.RevokeInviteCode(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, "revoked", resp["status"])
}

func TestRevokeInviteCode_NotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest("POST", "/api/invite-codes/00000000-0000-0000-0000-000000000000/revoke", nil)
	req.SetPathValue("id", "00000000-0000-0000-0000-000000000000")
	rr := httptest.NewRecorder()
	srv.RevokeInviteCode(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

// ---------------------------------------------------------------------------
// DELETE /api/invite-codes/{id}
// ---------------------------------------------------------------------------

func TestDeleteInviteCode_OK(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	ic, err := model.CreateInviteCode(context.Background(), db, "to-delete")
	require.NoError(t, err)

	req := httptest.NewRequest("DELETE", "/api/invite-codes/"+ic.ID, nil)
	req.SetPathValue("id", ic.ID)
	rr := httptest.NewRecorder()
	srv.DeleteInviteCode(rr, req)

	assert.Equal(t, http.StatusNoContent, rr.Code)

	// Confirm it's gone — validation should now return false.
	valid, err := model.ValidateAndRecordInviteCode(context.Background(), db, ic.Code, "127.0.0.1")
	require.NoError(t, err)
	assert.False(t, valid)
}

func TestDeleteInviteCode_NotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest("DELETE", "/api/invite-codes/00000000-0000-0000-0000-000000000000", nil)
	req.SetPathValue("id", "00000000-0000-0000-0000-000000000000")
	rr := httptest.NewRecorder()
	srv.DeleteInviteCode(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}
