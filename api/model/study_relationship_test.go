package model_test

import (
	"context"
	"testing"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateStudyRelationship(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	ctx := context.Background()

	studyA := testutil.CreateTestStudy(t, db, proj.ID)
	studyB := testutil.CreateTestStudy(t, db, proj.ID)

	rel, err := model.CreateStudyRelationship(ctx, db,
		studyA.ID, studyB.ID, "follow_up", "", "admin@example.com")
	require.NoError(t, err)
	assert.NotEmpty(t, rel.ID)
	assert.Equal(t, studyA.ID, rel.StudyID)
	assert.Equal(t, studyB.ID, rel.RelatedStudyID)
	assert.Equal(t, "follow_up", rel.Relationship)
	assert.Nil(t, rel.Notes) // empty string → nil
	assert.Equal(t, "admin@example.com", rel.CreatedBy)
	assert.False(t, rel.CreatedAt.IsZero())
}

func TestCreateStudyRelationship_WithNotes(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	ctx := context.Background()

	studyA := testutil.CreateTestStudy(t, db, proj.ID)
	studyB := testutil.CreateTestStudy(t, db, proj.ID)

	rel, err := model.CreateStudyRelationship(ctx, db,
		studyA.ID, studyB.ID, "baseline", "Baseline scan for longitudinal study", "admin@example.com")
	require.NoError(t, err)
	require.NotNil(t, rel.Notes)
	assert.Equal(t, "Baseline scan for longitudinal study", *rel.Notes)
}

func TestDeleteStudyRelationship(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	ctx := context.Background()

	studyA := testutil.CreateTestStudy(t, db, proj.ID)
	studyB := testutil.CreateTestStudy(t, db, proj.ID)

	rel, err := model.CreateStudyRelationship(ctx, db, studyA.ID, studyB.ID, "comparison", "", "user")
	require.NoError(t, err)

	deleted, err := model.DeleteStudyRelationship(ctx, db, rel.ID)
	require.NoError(t, err)
	assert.True(t, deleted)
}

func TestDeleteStudyRelationship_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)

	// Deleting a non-existent relationship returns false, not an error.
	deleted, err := model.DeleteStudyRelationship(context.Background(), db, "00000000-0000-0000-0000-000000000000")
	require.NoError(t, err)
	assert.False(t, deleted)
}

func TestListStudyRelationships_DirectLink(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	ctx := context.Background()

	studyA := testutil.CreateTestStudy(t, db, proj.ID)
	studyB := testutil.CreateTestStudy(t, db, proj.ID)

	_, err := model.CreateStudyRelationship(ctx, db, studyA.ID, studyB.ID, "follow_up", "", "user")
	require.NoError(t, err)

	// When listing from A's perspective, related study summary is B.
	rels, err := model.ListStudyRelationships(ctx, db, studyA.ID)
	require.NoError(t, err)
	require.Len(t, rels, 1)
	assert.Equal(t, studyB.ID, rels[0].RelatedStudy.ID)
	assert.Equal(t, "follow_up", rels[0].Relationship)
}

func TestListStudyRelationships_InverseLink(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	ctx := context.Background()

	studyA := testutil.CreateTestStudy(t, db, proj.ID)
	studyB := testutil.CreateTestStudy(t, db, proj.ID)

	_, err := model.CreateStudyRelationship(ctx, db, studyA.ID, studyB.ID, "baseline", "", "user")
	require.NoError(t, err)

	// When listing from B's perspective (the related_study_id side), related study summary is A.
	rels, err := model.ListStudyRelationships(ctx, db, studyB.ID)
	require.NoError(t, err)
	require.Len(t, rels, 1)
	assert.Equal(t, studyA.ID, rels[0].RelatedStudy.ID)
}

func TestListStudyRelationships_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)

	study := testutil.CreateTestStudy(t, db, proj.ID)

	rels, err := model.ListStudyRelationships(context.Background(), db, study.ID)
	require.NoError(t, err)
	// Returns empty slice (not nil) when no relationships exist.
	assert.NotNil(t, rels)
	assert.Len(t, rels, 0)
}

func TestListStudyRelationships_MultipleLinks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	ctx := context.Background()

	studyA := testutil.CreateTestStudy(t, db, proj.ID)
	studyB := testutil.CreateTestStudy(t, db, proj.ID)
	studyC := testutil.CreateTestStudy(t, db, proj.ID)

	_, err := model.CreateStudyRelationship(ctx, db, studyA.ID, studyB.ID, "follow_up", "", "user")
	require.NoError(t, err)
	_, err = model.CreateStudyRelationship(ctx, db, studyA.ID, studyC.ID, "comparison", "", "user")
	require.NoError(t, err)

	rels, err := model.ListStudyRelationships(ctx, db, studyA.ID)
	require.NoError(t, err)
	assert.Len(t, rels, 2)

	var relatedIDs []string
	for _, r := range rels {
		relatedIDs = append(relatedIDs, r.RelatedStudy.ID)
	}
	assert.Contains(t, relatedIDs, studyB.ID)
	assert.Contains(t, relatedIDs, studyC.ID)
}
