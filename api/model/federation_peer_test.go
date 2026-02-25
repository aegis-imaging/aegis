package model_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateFederationPeer(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	ts := time.Now().UnixNano()

	peer, err := model.CreateFederationPeer(context.Background(), db,
		fmt.Sprintf("Peer Create %d", ts),
		fmt.Sprintf("peer-create-%d", ts),
		"https://peer.example.com",
		"test peer",
	)
	require.NoError(t, err)
	assert.NotEmpty(t, peer.ID)
	assert.Equal(t, "https://peer.example.com", peer.APIURL)
	assert.Equal(t, "test peer", peer.Notes)
	assert.True(t, peer.Enabled) // default enabled
}

func TestGetFederationPeer(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	ts := time.Now().UnixNano()
	ctx := context.Background()

	peer, err := model.CreateFederationPeer(ctx, db,
		fmt.Sprintf("Peer Get %d", ts),
		fmt.Sprintf("peer-get-%d", ts),
		"https://get-peer.example.com",
		"notes here",
	)
	require.NoError(t, err)

	got, err := model.GetFederationPeer(ctx, db, peer.ID)
	require.NoError(t, err)
	assert.Equal(t, peer.ID, got.ID)
	assert.Equal(t, peer.APIURL, got.APIURL)
}

func TestGetFederationPeer_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)

	_, err := model.GetFederationPeer(context.Background(), db, "00000000-0000-0000-0000-000000000000")
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

func TestListFederationPeers(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	ts := time.Now().UnixNano()
	ctx := context.Background()

	p1, err := model.CreateFederationPeer(ctx, db,
		fmt.Sprintf("Alpha Fed %d", ts), fmt.Sprintf("alpha-fed-%d", ts), "https://a.example.com", "")
	require.NoError(t, err)

	p2, err := model.CreateFederationPeer(ctx, db,
		fmt.Sprintf("Beta Fed %d", ts), fmt.Sprintf("beta-fed-%d", ts), "https://b.example.com", "")
	require.NoError(t, err)

	peers, err := model.ListFederationPeers(ctx, db)
	require.NoError(t, err)

	var ids []string
	for _, p := range peers {
		ids = append(ids, p.ID)
	}
	assert.Contains(t, ids, p1.ID)
	assert.Contains(t, ids, p2.ID)
}

func TestUpdateFederationPeer(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	ts := time.Now().UnixNano()
	ctx := context.Background()

	peer, err := model.CreateFederationPeer(ctx, db,
		fmt.Sprintf("Update Fed %d", ts),
		fmt.Sprintf("update-fed-%d", ts),
		"https://old.example.com",
		"",
	)
	require.NoError(t, err)

	updated, err := model.UpdateFederationPeer(ctx, db,
		peer.ID,
		fmt.Sprintf("Updated Fed %d", ts),
		fmt.Sprintf("update-fed-%d", ts), // same slug is fine
		"https://new.example.com",
		"updated notes",
		false, // disabled
	)
	require.NoError(t, err)
	assert.Equal(t, peer.ID, updated.ID)
	assert.Equal(t, "https://new.example.com", updated.APIURL)
	assert.Equal(t, "updated notes", updated.Notes)
	assert.False(t, updated.Enabled)
}

func TestDeleteFederationPeer(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	ts := time.Now().UnixNano()
	ctx := context.Background()

	peer, err := model.CreateFederationPeer(ctx, db,
		fmt.Sprintf("Delete Fed %d", ts),
		fmt.Sprintf("delete-fed-%d", ts),
		"https://delete.example.com",
		"",
	)
	require.NoError(t, err)

	require.NoError(t, model.DeleteFederationPeer(ctx, db, peer.ID))

	_, err = model.GetFederationPeer(ctx, db, peer.ID)
	assert.ErrorIs(t, err, sql.ErrNoRows)
}
