package model_test

import (
	"context"
	"testing"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetProjectPhiConfig_Defaults(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)

	cfg, err := model.GetProjectPhiConfig(context.Background(), db, proj.ID)
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, proj.ID, cfg.ProjectID)
	// Default values when no override is set.
	assert.InDelta(t, 0.4, cfg.ConfidenceThreshold, 0.001)
	assert.Equal(t, 3, cfg.MinTextLength)
}

func TestUpsertProjectPhiConfig_Create(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	ctx := context.Background()

	cfg, err := model.UpsertProjectPhiConfig(ctx, db, proj.ID, 0.7, 5)
	require.NoError(t, err)
	assert.Equal(t, proj.ID, cfg.ProjectID)
	assert.InDelta(t, 0.7, cfg.ConfidenceThreshold, 0.001)
	assert.Equal(t, 5, cfg.MinTextLength)
}

func TestUpsertProjectPhiConfig_Update(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	ctx := context.Background()

	// Create initial override.
	_, err := model.UpsertProjectPhiConfig(ctx, db, proj.ID, 0.6, 4)
	require.NoError(t, err)

	// Upsert again — should update, not insert a duplicate.
	updated, err := model.UpsertProjectPhiConfig(ctx, db, proj.ID, 0.9, 8)
	require.NoError(t, err)
	assert.InDelta(t, 0.9, updated.ConfidenceThreshold, 0.001)
	assert.Equal(t, 8, updated.MinTextLength)

	// GetProjectPhiConfig should return the updated values.
	got, err := model.GetProjectPhiConfig(ctx, db, proj.ID)
	require.NoError(t, err)
	assert.InDelta(t, 0.9, got.ConfidenceThreshold, 0.001)
	assert.Equal(t, 8, got.MinTextLength)
}

func TestGetProjectPhiConfig_AfterUpsert(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.CreateTestProject(t, db, "phi-config-proj")
	ctx := context.Background()

	_, err := model.UpsertProjectPhiConfig(ctx, db, proj.ID, 0.85, 6)
	require.NoError(t, err)

	got, err := model.GetProjectPhiConfig(ctx, db, proj.ID)
	require.NoError(t, err)
	assert.InDelta(t, 0.85, got.ConfidenceThreshold, 0.001)
	assert.Equal(t, 6, got.MinTextLength)
	assert.Equal(t, proj.ID, got.ProjectID)
}
