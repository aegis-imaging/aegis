package model_test

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func hashKey(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

func TestCreateAPIKey(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	ctx := context.Background()

	rawKey := fmt.Sprintf("aegis_test_%d", time.Now().UnixNano())
	keyHash := hashKey(rawKey)
	prefix := rawKey[:8]

	k, err := model.CreateAPIKey(ctx, db, "ci-key", keyHash, prefix, "ci@test.com", nil)
	require.NoError(t, err)
	assert.NotEmpty(t, k.ID)
	assert.Equal(t, "ci-key", k.Name)
	assert.Equal(t, prefix, k.KeyPrefix)
	assert.Equal(t, "ci@test.com", k.CreatedBy)
	assert.True(t, k.Enabled)
	assert.Nil(t, k.ExpiresAt)
	assert.Nil(t, k.LastUsedAt)
}

func TestCreateAPIKey_WithExpiry(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	ctx := context.Background()

	rawKey := fmt.Sprintf("aegis_expiring_%d", time.Now().UnixNano())
	keyHash := hashKey(rawKey)
	exp := time.Now().UTC().Add(24 * time.Hour)

	k, err := model.CreateAPIKey(ctx, db, "expiring-key", keyHash, rawKey[:8], "ci@test.com", &exp)
	require.NoError(t, err)
	require.NotNil(t, k.ExpiresAt)
	assert.WithinDuration(t, exp, *k.ExpiresAt, time.Second)
}

func TestGetAPIKeyByHash(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	ctx := context.Background()

	rawKey := fmt.Sprintf("aegis_hash_lookup_%d", time.Now().UnixNano())
	keyHash := hashKey(rawKey)

	created, err := model.CreateAPIKey(ctx, db, "hash-test", keyHash, rawKey[:8], "test@test.com", nil)
	require.NoError(t, err)

	got, err := model.GetAPIKeyByHash(ctx, db, keyHash)
	require.NoError(t, err)
	assert.Equal(t, created.ID, got.ID)
	assert.Equal(t, "hash-test", got.Name)
}

func TestGetAPIKeyByHash_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)

	_, err := model.GetAPIKeyByHash(context.Background(), db, "nonexistenthash")
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

func TestListAPIKeys(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	ctx := context.Background()

	r1 := fmt.Sprintf("aegis_list_a_%d", time.Now().UnixNano())
	r2 := fmt.Sprintf("aegis_list_b_%d", time.Now().UnixNano()+1)
	_, err := model.CreateAPIKey(ctx, db, "key-alpha", hashKey(r1), r1[:8], "me", nil)
	require.NoError(t, err)
	_, err = model.CreateAPIKey(ctx, db, "key-beta", hashKey(r2), r2[:8], "me", nil)
	require.NoError(t, err)

	keys, err := model.ListAPIKeys(ctx, db)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(keys), 2)
}

func TestUpdateAPIKeyEnabled(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	ctx := context.Background()

	rawKey := fmt.Sprintf("aegis_enable_%d", time.Now().UnixNano())
	k, err := model.CreateAPIKey(ctx, db, "toggle-key", hashKey(rawKey), rawKey[:8], "me", nil)
	require.NoError(t, err)
	assert.True(t, k.Enabled)

	require.NoError(t, model.UpdateAPIKeyEnabled(ctx, db, k.ID, false))

	got, err := model.GetAPIKeyByHash(ctx, db, hashKey(rawKey))
	require.NoError(t, err)
	assert.False(t, got.Enabled)

	require.NoError(t, model.UpdateAPIKeyEnabled(ctx, db, k.ID, true))
	got, err = model.GetAPIKeyByHash(ctx, db, hashKey(rawKey))
	require.NoError(t, err)
	assert.True(t, got.Enabled)
}

func TestRotateAPIKey(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	ctx := context.Background()

	oldRaw := fmt.Sprintf("aegis_old_%d", time.Now().UnixNano())
	k, err := model.CreateAPIKey(ctx, db, "rotate-key", hashKey(oldRaw), oldRaw[:8], "me", nil)
	require.NoError(t, err)

	newRaw := fmt.Sprintf("aegis_new_%d", time.Now().UnixNano())
	newHash := hashKey(newRaw)
	rotated, err := model.RotateAPIKey(ctx, db, k.ID, newHash, newRaw[:8])
	require.NoError(t, err)
	assert.Equal(t, k.ID, rotated.ID)
	assert.Equal(t, newRaw[:8], rotated.KeyPrefix)

	// Old hash should no longer resolve.
	_, err = model.GetAPIKeyByHash(ctx, db, hashKey(oldRaw))
	assert.ErrorIs(t, err, sql.ErrNoRows)

	// New hash should resolve.
	got, err := model.GetAPIKeyByHash(ctx, db, newHash)
	require.NoError(t, err)
	assert.Equal(t, k.ID, got.ID)
}

func TestDeleteAPIKey(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	ctx := context.Background()

	rawKey := fmt.Sprintf("aegis_del_%d", time.Now().UnixNano())
	k, err := model.CreateAPIKey(ctx, db, "delete-me", hashKey(rawKey), rawKey[:8], "me", nil)
	require.NoError(t, err)

	require.NoError(t, model.DeleteAPIKey(ctx, db, k.ID))

	_, err = model.GetAPIKeyByHash(ctx, db, hashKey(rawKey))
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

func TestTouchAPIKey(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	ctx := context.Background()

	rawKey := fmt.Sprintf("aegis_touch_%d", time.Now().UnixNano())
	k, err := model.CreateAPIKey(ctx, db, "touch-key", hashKey(rawKey), rawKey[:8], "me", nil)
	require.NoError(t, err)
	assert.Nil(t, k.LastUsedAt)

	model.TouchAPIKey(ctx, db, k.ID)

	got, err := model.GetAPIKeyByHash(ctx, db, hashKey(rawKey))
	require.NoError(t, err)
	assert.NotNil(t, got.LastUsedAt, "TouchAPIKey should set last_used_at")
}
