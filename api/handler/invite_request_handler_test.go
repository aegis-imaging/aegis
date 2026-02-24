package handler_test

import (
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

// ─── ListInviteRequestsAdmin ─────────────────────────────────────────────────

func TestListInviteRequests_Empty(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/invite/requests", nil)
	rr := httptest.NewRecorder()
	srv.ListInviteRequestsAdmin(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var body struct {
		Requests []any `json:"requests"`
		Total    int   `json:"total"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&body))
	assert.Empty(t, body.Requests)
	assert.Equal(t, 0, body.Total)
}

func TestListInviteRequests_ReturnsPending(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	_, err := model.CreateInviteRequest(context.Background(), db, "Alice", "alice@example.com", "", "", "1.2.3.4")
	require.NoError(t, err)
	_, err = model.CreateInviteRequest(context.Background(), db, "Bob", "bob@example.com", "", "", "1.2.3.5")
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/invite/requests?status=pending", nil)
	rr := httptest.NewRecorder()
	srv.ListInviteRequestsAdmin(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var body struct {
		Requests []map[string]any `json:"requests"`
		Total    int              `json:"total"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&body))
	assert.Equal(t, 2, body.Total)
	assert.Len(t, body.Requests, 2)
}

func TestListInviteRequests_InvalidStatus(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/invite/requests?status=unknown", nil)
	rr := httptest.NewRecorder()
	srv.ListInviteRequestsAdmin(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

// ─── ApproveInviteRequestAdmin ───────────────────────────────────────────────

func TestApproveInviteRequestAdmin_NotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodPost, "/api/invite/requests/00000000-0000-0000-0000-000000000000/approve", nil)
	req.SetPathValue("id", "00000000-0000-0000-0000-000000000000")
	rr := httptest.NewRecorder()
	srv.ApproveInviteRequestAdmin(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestApproveInviteRequestAdmin_AlreadyApproved(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	ir, err := model.CreateInviteRequest(context.Background(), db, "Carol", "carol@example.com", "", "", "")
	require.NoError(t, err)

	ic, err := model.CreateInviteCode(context.Background(), db, "carol label")
	require.NoError(t, err)
	require.NoError(t, model.ApproveInviteRequest(context.Background(), db, ir.ID, "admin@example.com", ic.ID))

	req := httptest.NewRequest(http.MethodPost, "/api/invite/requests/"+ir.ID+"/approve", nil)
	req.SetPathValue("id", ir.ID)
	rr := httptest.NewRecorder()
	srv.ApproveInviteRequestAdmin(rr, req)

	assert.Equal(t, http.StatusConflict, rr.Code)
	assert.Contains(t, rr.Body.String(), "not pending")
}

func TestApproveInviteRequestAdmin_Success(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	ir, err := model.CreateInviteRequest(context.Background(), db, "Dave", "dave@example.com", "Corp", "hi", "")
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/invite/requests/"+ir.ID+"/approve", nil)
	req.SetPathValue("id", ir.ID)
	rr := httptest.NewRecorder()
	srv.ApproveInviteRequestAdmin(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var body struct {
		Status     string         `json:"status"`
		InviteCode map[string]any `json:"invite_code"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&body))
	assert.Equal(t, "approved", body.Status)
	assert.NotEmpty(t, body.InviteCode["code"])

	updated, err := model.GetInviteRequest(context.Background(), db, ir.ID)
	require.NoError(t, err)
	assert.Equal(t, "approved", updated.Status)
	assert.NotNil(t, updated.InviteCodeID)
}

// ─── DenyInviteRequestAdmin ──────────────────────────────────────────────────

func TestDenyInviteRequestAdmin_NotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodPost, "/api/invite/requests/00000000-0000-0000-0000-000000000000/deny", nil)
	req.SetPathValue("id", "00000000-0000-0000-0000-000000000000")
	rr := httptest.NewRecorder()
	srv.DenyInviteRequestAdmin(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestDenyInviteRequestAdmin_AlreadyDenied(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	ir, err := model.CreateInviteRequest(context.Background(), db, "Eve", "eve@example.com", "", "", "")
	require.NoError(t, err)
	require.NoError(t, model.DenyInviteRequest(context.Background(), db, ir.ID, "admin@example.com"))

	req := httptest.NewRequest(http.MethodPost, "/api/invite/requests/"+ir.ID+"/deny", nil)
	req.SetPathValue("id", ir.ID)
	rr := httptest.NewRecorder()
	srv.DenyInviteRequestAdmin(rr, req)

	assert.Equal(t, http.StatusConflict, rr.Code)
	assert.Contains(t, rr.Body.String(), "not pending")
}

func TestDenyInviteRequestAdmin_Success(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	ir, err := model.CreateInviteRequest(context.Background(), db, "Frank", "frank@example.com", "", "", "")
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/invite/requests/"+ir.ID+"/deny", nil)
	req.SetPathValue("id", ir.ID)
	rr := httptest.NewRecorder()
	srv.DenyInviteRequestAdmin(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var body struct{ Status string `json:"status"` }
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&body))
	assert.Equal(t, "denied", body.Status)

	updated, err := model.GetInviteRequest(context.Background(), db, ir.ID)
	require.NoError(t, err)
	assert.Equal(t, "denied", updated.Status)
}
