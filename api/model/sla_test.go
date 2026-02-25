package model_test

import (
	"context"
	"testing"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetStuckStudies_NoStuck(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	// Create a fresh study — updated_at = now, should NOT be stuck with 60-minute threshold.
	testutil.CreateTestStudy(t, db, proj.ID)

	studies, err := model.GetStuckStudies(context.Background(), db, 60, "")
	require.NoError(t, err)
	assert.Empty(t, studies, "fresh study should not appear as stuck")
}

func TestGetStuckStudies_Stuck(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	ctx := context.Background()

	study := testutil.CreateTestStudy(t, db, proj.ID)

	// Manually backdate updated_at to simulate a stuck study.
	_, err := db.ExecContext(ctx,
		`UPDATE studies SET updated_at = NOW() - INTERVAL '120 minutes' WHERE id = $1`, study.ID)
	require.NoError(t, err)

	studies, err := model.GetStuckStudies(ctx, db, 60, "")
	require.NoError(t, err)

	var ids []string
	for _, s := range studies {
		ids = append(ids, s.ID)
	}
	assert.Contains(t, ids, study.ID, "backdated study should appear as stuck")
}

func TestGetStuckStudies_ApprovedNotStuck(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	ctx := context.Background()

	study := testutil.CreateTestStudy(t, db, proj.ID)
	// Set to approved (terminal) and backdate.
	_, err := db.ExecContext(ctx,
		`UPDATE studies SET status = 'approved', updated_at = NOW() - INTERVAL '120 minutes' WHERE id = $1`,
		study.ID)
	require.NoError(t, err)

	studies, err := model.GetStuckStudies(ctx, db, 60, "")
	require.NoError(t, err)

	for _, s := range studies {
		assert.NotEqual(t, study.ID, s.ID, "approved study should not appear as stuck")
	}
}

func TestGetStuckStudies_ProjectFilter(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	otherProj := testutil.CreateTestProject(t, db, "sla-other-project")
	ctx := context.Background()

	stuckInProj := testutil.CreateTestStudy(t, db, proj.ID)
	stuckInOther := testutil.CreateTestStudy(t, db, otherProj.ID)

	for _, id := range []string{stuckInProj.ID, stuckInOther.ID} {
		_, err := db.ExecContext(ctx,
			`UPDATE studies SET updated_at = NOW() - INTERVAL '120 minutes' WHERE id = $1`, id)
		require.NoError(t, err)
	}

	studies, err := model.GetStuckStudies(ctx, db, 60, proj.ID)
	require.NoError(t, err)

	var ids []string
	for _, s := range studies {
		ids = append(ids, s.ID)
	}
	assert.Contains(t, ids, stuckInProj.ID)
	assert.NotContains(t, ids, stuckInOther.ID, "study from other project should not appear")
}

func TestMarkStudySLAAlerted_AndUnalerted(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	ctx := context.Background()

	stuckStudy := testutil.CreateTestStudy(t, db, proj.ID)
	freshStudy := testutil.CreateTestStudy(t, db, proj.ID)

	for _, id := range []string{stuckStudy.ID, freshStudy.ID} {
		_, err := db.ExecContext(ctx,
			`UPDATE studies SET updated_at = NOW() - INTERVAL '120 minutes' WHERE id = $1`, id)
		require.NoError(t, err)
	}

	// Both should appear as unalerted with a 1-hour cooldown.
	unalerted, err := model.GetUnalertedStuckStudies(ctx, db, 60, 1)
	require.NoError(t, err)
	var preAlertIDs []string
	for _, s := range unalerted {
		preAlertIDs = append(preAlertIDs, s.ID)
	}
	assert.Contains(t, preAlertIDs, stuckStudy.ID)
	assert.Contains(t, preAlertIDs, freshStudy.ID)

	// Mark stuckStudy as alerted.
	require.NoError(t, model.MarkStudySLAAlerted(ctx, db, stuckStudy.ID))

	// Re-query with a 1-hour cooldown — stuckStudy was just alerted so it should be suppressed.
	unalerted2, err := model.GetUnalertedStuckStudies(ctx, db, 60, 1)
	require.NoError(t, err)
	var afterIDs []string
	for _, s := range unalerted2 {
		afterIDs = append(afterIDs, s.ID)
	}
	assert.NotContains(t, afterIDs, stuckStudy.ID, "just-alerted study should be suppressed by cooldown")
	assert.Contains(t, afterIDs, freshStudy.ID)
}

func TestMarkStudySLAAlerted_Idempotent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	ctx := context.Background()

	study := testutil.CreateTestStudy(t, db, proj.ID)

	// Calling MarkStudySLAAlerted twice should not error.
	require.NoError(t, model.MarkStudySLAAlerted(ctx, db, study.ID))
	require.NoError(t, model.MarkStudySLAAlerted(ctx, db, study.ID))
}
