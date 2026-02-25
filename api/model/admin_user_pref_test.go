package model_test

import (
	"context"
	"testing"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAdminUserPreferences_Defaults(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	u := testutil.CreateTestAdminUser(t, db, "prefs-default@example.com", "admin")

	prefs, err := model.GetAdminUserPreferences(context.Background(), db, u.ID)
	require.NoError(t, err)
	assert.Equal(t, u.ID, prefs.AdminUserID)
	// No record stored → default values.
	assert.Equal(t, "weekly", prefs.DigestFrequency)
	assert.Equal(t, []string{}, prefs.NotifyEvents)
}

func TestUpsertAdminUserPreferences_Create(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	u := testutil.CreateTestAdminUser(t, db, "prefs-create@example.com", "admin")
	ctx := context.Background()

	p := &model.AdminUserPreferences{
		AdminUserID:     u.ID,
		DigestFrequency: "monthly",
		NotifyEvents:    []string{"study.approved", "study.stuck"},
	}
	require.NoError(t, model.UpsertAdminUserPreferences(ctx, db, p))

	got, err := model.GetAdminUserPreferences(ctx, db, u.ID)
	require.NoError(t, err)
	assert.Equal(t, "monthly", got.DigestFrequency)
	assert.ElementsMatch(t, []string{"study.approved", "study.stuck"}, got.NotifyEvents)
}

func TestUpsertAdminUserPreferences_Update(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	u := testutil.CreateTestAdminUser(t, db, "prefs-update@example.com", "admin")
	ctx := context.Background()

	// Create initial preferences.
	first := &model.AdminUserPreferences{
		AdminUserID:     u.ID,
		DigestFrequency: "daily",
		NotifyEvents:    []string{"pipeline.failed"},
	}
	require.NoError(t, model.UpsertAdminUserPreferences(ctx, db, first))

	// Update to different values via upsert.
	second := &model.AdminUserPreferences{
		AdminUserID:     u.ID,
		DigestFrequency: "none",
		NotifyEvents:    []string{"study.phi_flagged", "study.rejected"},
	}
	require.NoError(t, model.UpsertAdminUserPreferences(ctx, db, second))

	got, err := model.GetAdminUserPreferences(ctx, db, u.ID)
	require.NoError(t, err)
	assert.Equal(t, "none", got.DigestFrequency)
	assert.ElementsMatch(t, []string{"study.phi_flagged", "study.rejected"}, got.NotifyEvents)
}

func TestUpsertAdminUserPreferences_EmptyEvents(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	u := testutil.CreateTestAdminUser(t, db, "prefs-empty@example.com", "admin")
	ctx := context.Background()

	p := &model.AdminUserPreferences{
		AdminUserID:     u.ID,
		DigestFrequency: "weekly",
		NotifyEvents:    []string{},
	}
	require.NoError(t, model.UpsertAdminUserPreferences(ctx, db, p))

	got, err := model.GetAdminUserPreferences(ctx, db, u.ID)
	require.NoError(t, err)
	assert.Equal(t, []string{}, got.NotifyEvents)
}
