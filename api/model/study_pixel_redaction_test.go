package model_test

import (
	"context"
	"testing"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetPixelRedactionRequired(t *testing.T) {
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	// Initially not required
	assert.False(t, study.PixelRedactionRequired)
	assert.Empty(t, study.PixelRedactionStatus)

	// Set required
	err := model.SetPixelRedactionRequired(context.Background(), db, study.ID, true)
	require.NoError(t, err)

	updated, err := model.GetStudyByID(context.Background(), db, study.ID)
	require.NoError(t, err)
	assert.True(t, updated.PixelRedactionRequired)
	assert.Equal(t, "pending", updated.PixelRedactionStatus)
}

func TestUpdatePixelRedactionStatus(t *testing.T) {
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	require.NoError(t, model.SetPixelRedactionRequired(context.Background(), db, study.ID, true))

	// Transition pending → redacting
	err := model.UpdatePixelRedactionStatus(context.Background(), db, study.ID, "redacting")
	require.NoError(t, err)

	updated, err := model.GetStudyByID(context.Background(), db, study.ID)
	require.NoError(t, err)
	assert.Equal(t, "redacting", updated.PixelRedactionStatus)

	// Transition redacting → complete
	err = model.UpdatePixelRedactionStatus(context.Background(), db, study.ID, "complete")
	require.NoError(t, err)

	updated, err = model.GetStudyByID(context.Background(), db, study.ID)
	require.NoError(t, err)
	assert.Equal(t, "complete", updated.PixelRedactionStatus)
}

func TestClaimPixelRedaction(t *testing.T) {
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	require.NoError(t, model.SetPixelRedactionRequired(context.Background(), db, study.ID, true))

	// First claim should succeed
	claimed, err := model.ClaimPixelRedaction(context.Background(), db, study.ID)
	require.NoError(t, err)
	assert.True(t, claimed)

	// Second claim should fail (already redacting)
	claimed, err = model.ClaimPixelRedaction(context.Background(), db, study.ID)
	require.NoError(t, err)
	assert.False(t, claimed)

	// Verify status is "redacting"
	updated, err := model.GetStudyByID(context.Background(), db, study.ID)
	require.NoError(t, err)
	assert.Equal(t, "redacting", updated.PixelRedactionStatus)
}
