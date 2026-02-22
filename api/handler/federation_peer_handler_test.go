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

func TestListFederationPeers_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest("GET", "/api/federation-peers", nil)
	rr := httptest.NewRecorder()
	srv.ListFederationPeers(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var peers []model.FederationPeer
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&peers))
	assert.NotNil(t, peers)
	assert.Len(t, peers, 0)
}

func TestCreateFederationPeer_OK(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body, _ := json.Marshal(map[string]string{
		"name":    "Partner Institute",
		"api_url": "https://peer.example.com",
		"notes":   "test peer",
	})
	req := httptest.NewRequest("POST", "/api/federation-peers", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	srv.CreateFederationPeer(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)
	var peer model.FederationPeer
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&peer))
	assert.NotEmpty(t, peer.ID)
	assert.Equal(t, "Partner Institute", peer.Name)
	assert.Equal(t, "partner-institute", peer.Slug)
	assert.Equal(t, "https://peer.example.com", peer.APIURL)
	assert.True(t, peer.Enabled)
}

func TestCreateFederationPeer_MissingName(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body, _ := json.Marshal(map[string]string{"api_url": "https://peer.example.com"})
	req := httptest.NewRequest("POST", "/api/federation-peers", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	srv.CreateFederationPeer(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestCreateFederationPeer_MissingAPIURL(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body, _ := json.Marshal(map[string]string{"name": "Test Peer"})
	req := httptest.NewRequest("POST", "/api/federation-peers", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	srv.CreateFederationPeer(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestGetFederationPeer_OK(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	peer, err := model.CreateFederationPeer(
		context.Background(), db, "Get Test Peer", "get-test-peer", "https://peer.example.com", "")
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/api/federation-peers/"+peer.ID, nil)
	req.SetPathValue("id", peer.ID)
	rr := httptest.NewRecorder()
	srv.GetFederationPeer(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var got model.FederationPeer
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&got))
	assert.Equal(t, peer.ID, got.ID)
}

func TestGetFederationPeer_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest("GET", "/api/federation-peers/nonexistent", nil)
	req.SetPathValue("id", "nonexistent")
	rr := httptest.NewRecorder()
	srv.GetFederationPeer(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestUpdateFederationPeer_OK(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	peer, err := model.CreateFederationPeer(
		context.Background(), db, "Update Test Peer", "update-test-peer", "https://peer.example.com", "")
	require.NoError(t, err)

	body, _ := json.Marshal(map[string]any{
		"name":    "Updated Peer",
		"api_url": "https://updated.example.com",
		"enabled": false,
	})
	req := httptest.NewRequest("PUT", "/api/federation-peers/"+peer.ID, bytes.NewReader(body))
	req.SetPathValue("id", peer.ID)
	rr := httptest.NewRecorder()
	srv.UpdateFederationPeer(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var updated model.FederationPeer
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&updated))
	assert.Equal(t, "Updated Peer", updated.Name)
	assert.False(t, updated.Enabled)
}

func TestDeleteFederationPeer_OK(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	peer, err := model.CreateFederationPeer(
		context.Background(), db, "Delete Test Peer", "delete-test-peer", "https://peer.example.com", "")
	require.NoError(t, err)

	req := httptest.NewRequest("DELETE", "/api/federation-peers/"+peer.ID, nil)
	req.SetPathValue("id", peer.ID)
	rr := httptest.NewRecorder()
	srv.DeleteFederationPeer(rr, req)

	assert.Equal(t, http.StatusNoContent, rr.Code)

	// Verify it's gone
	req2 := httptest.NewRequest("GET", "/api/federation-peers/"+peer.ID, nil)
	req2.SetPathValue("id", peer.ID)
	rr2 := httptest.NewRecorder()
	srv.GetFederationPeer(rr2, req2)
	assert.Equal(t, http.StatusNotFound, rr2.Code)
}

func TestFederationPeer_SlugConflict(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body, _ := json.Marshal(map[string]string{
		"name":    "Conflict Peer",
		"slug":    "conflict-peer",
		"api_url": "https://peer1.example.com",
	})
	req := httptest.NewRequest("POST", "/api/federation-peers", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	srv.CreateFederationPeer(rr, req)
	assert.Equal(t, http.StatusCreated, rr.Code)

	// Same slug again
	body2, _ := json.Marshal(map[string]string{
		"name":    "Conflict Peer 2",
		"slug":    "conflict-peer",
		"api_url": "https://peer2.example.com",
	})
	req2 := httptest.NewRequest("POST", "/api/federation-peers", bytes.NewReader(body2))
	rr2 := httptest.NewRecorder()
	srv.CreateFederationPeer(rr2, req2)
	assert.Equal(t, http.StatusConflict, rr2.Code)
}
