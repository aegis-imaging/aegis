package model_test

import (
	"context"
	"testing"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddStudyLabel(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)
	ctx := context.Background()

	l, err := model.AddStudyLabel(ctx, db, study.ID, "cohort-A", "admin@test.com")
	require.NoError(t, err)
	assert.NotEmpty(t, l.ID)
	assert.Equal(t, study.ID, l.StudyID)
	assert.Equal(t, "cohort-A", l.Label)
	assert.Equal(t, "admin@test.com", l.CreatedBy)
}

func TestAddStudyLabel_Duplicate_CaseInsensitive(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)
	ctx := context.Background()

	l1, err := model.AddStudyLabel(ctx, db, study.ID, "Cohort-A", "admin@test.com")
	require.NoError(t, err)

	// Adding the same label with different casing should return the existing one.
	l2, err := model.AddStudyLabel(ctx, db, study.ID, "cohort-a", "other@test.com")
	require.NoError(t, err)
	assert.Equal(t, l1.ID, l2.ID, "duplicate label should return the existing record")

	// Only one label should exist for the study.
	labels, err := model.ListStudyLabels(ctx, db, study.ID)
	require.NoError(t, err)
	assert.Len(t, labels, 1)
}

func TestListStudyLabels(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)
	ctx := context.Background()

	_, err := model.AddStudyLabel(ctx, db, study.ID, "alpha", "me")
	require.NoError(t, err)
	_, err = model.AddStudyLabel(ctx, db, study.ID, "beta", "me")
	require.NoError(t, err)

	labels, err := model.ListStudyLabels(ctx, db, study.ID)
	require.NoError(t, err)
	assert.Len(t, labels, 2)
	// Ordered by created_at ASC — alpha was inserted first.
	assert.Equal(t, "alpha", labels[0].Label)
	assert.Equal(t, "beta", labels[1].Label)
}

func TestDeleteStudyLabel(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)
	ctx := context.Background()

	l, err := model.AddStudyLabel(ctx, db, study.ID, "to-delete", "me")
	require.NoError(t, err)

	require.NoError(t, model.DeleteStudyLabel(ctx, db, l.ID, study.ID))

	labels, err := model.ListStudyLabels(ctx, db, study.ID)
	require.NoError(t, err)
	assert.Empty(t, labels)
}

func TestDeleteStudyLabel_WrongStudy_NoOp(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)
	other := testutil.CreateTestStudy(t, db, proj.ID)
	ctx := context.Background()

	l, err := model.AddStudyLabel(ctx, db, study.ID, "keep-me", "me")
	require.NoError(t, err)

	// Attempting to delete with the wrong study ID should be a no-op.
	require.NoError(t, model.DeleteStudyLabel(ctx, db, l.ID, other.ID))

	labels, err := model.ListStudyLabels(ctx, db, study.ID)
	require.NoError(t, err)
	assert.Len(t, labels, 1, "label should still exist")
}

func TestBulkAddLabel(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	s1 := testutil.CreateTestStudy(t, db, proj.ID)
	s2 := testutil.CreateTestStudy(t, db, proj.ID)
	s3 := testutil.CreateTestStudy(t, db, proj.ID)
	ctx := context.Background()

	n, err := model.BulkAddLabel(ctx, db, []string{s1.ID, s2.ID, s3.ID}, "pilot", "me")
	require.NoError(t, err)
	assert.Equal(t, int64(3), n)

	// Each study should have the label.
	for _, sid := range []string{s1.ID, s2.ID, s3.ID} {
		labels, err := model.ListStudyLabels(ctx, db, sid)
		require.NoError(t, err)
		require.Len(t, labels, 1)
		assert.Equal(t, "pilot", labels[0].Label)
	}
}

func TestBulkAddLabel_Idempotent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)
	ctx := context.Background()

	// Add once.
	n1, err := model.BulkAddLabel(ctx, db, []string{study.ID}, "idempotent", "me")
	require.NoError(t, err)
	assert.Equal(t, int64(1), n1)

	// Add again — conflict ignored, 0 rows inserted.
	n2, err := model.BulkAddLabel(ctx, db, []string{study.ID}, "idempotent", "me")
	require.NoError(t, err)
	assert.Equal(t, int64(0), n2)

	// Still only one label on the study.
	labels, err := model.ListStudyLabels(ctx, db, study.ID)
	require.NoError(t, err)
	assert.Len(t, labels, 1)
}

func TestBulkRemoveLabel(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	s1 := testutil.CreateTestStudy(t, db, proj.ID)
	s2 := testutil.CreateTestStudy(t, db, proj.ID)
	ctx := context.Background()

	_, err := model.BulkAddLabel(ctx, db, []string{s1.ID, s2.ID}, "remove-me", "me")
	require.NoError(t, err)

	n, err := model.BulkRemoveLabel(ctx, db, []string{s1.ID, s2.ID}, "remove-me")
	require.NoError(t, err)
	assert.Equal(t, int64(2), n)

	for _, sid := range []string{s1.ID, s2.ID} {
		labels, err := model.ListStudyLabels(ctx, db, sid)
		require.NoError(t, err)
		assert.Empty(t, labels)
	}
}

func TestBulkRemoveLabel_CaseInsensitive(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)
	ctx := context.Background()

	_, err := model.AddStudyLabel(ctx, db, study.ID, "UPPER-CASE", "me")
	require.NoError(t, err)

	// Remove using lowercase — should still match.
	n, err := model.BulkRemoveLabel(ctx, db, []string{study.ID}, "upper-case")
	require.NoError(t, err)
	assert.Equal(t, int64(1), n)

	labels, err := model.ListStudyLabels(ctx, db, study.ID)
	require.NoError(t, err)
	assert.Empty(t, labels)
}
