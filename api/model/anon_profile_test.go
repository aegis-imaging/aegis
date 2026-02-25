package model_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateAnonProfile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)

	tags, _ := json.Marshal([]string{"PatientAge", "StudyDate"})
	p := &model.AnonProfile{
		ProjectID:    proj.ID,
		Name:         "brain-mri",
		Description:  "Brain MRI profile",
		RetainedTags: tags,
		Enabled:      true,
	}

	require.NoError(t, model.CreateAnonProfile(context.Background(), db, p))
	assert.NotEmpty(t, p.ID)
	assert.False(t, p.CreatedAt.IsZero())
}

func TestCreateAnonProfile_NilTagsDefaultsToEmpty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)

	p := &model.AnonProfile{
		ProjectID: proj.ID,
		Name:      "nil-tags-profile",
		Enabled:   true,
		// RetainedTags is nil
	}

	require.NoError(t, model.CreateAnonProfile(context.Background(), db, p))
	assert.Equal(t, json.RawMessage("[]"), p.RetainedTags)
}

func TestGetAnonProfileByID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)

	p := &model.AnonProfile{
		ProjectID: proj.ID,
		Name:      "get-by-id",
		Enabled:   true,
	}
	require.NoError(t, model.CreateAnonProfile(context.Background(), db, p))

	got, err := model.GetAnonProfileByID(context.Background(), db, p.ID)
	require.NoError(t, err)
	assert.Equal(t, p.ID, got.ID)
	assert.Equal(t, "get-by-id", got.Name)
	assert.Equal(t, proj.ID, got.ProjectID)
}

func TestListAnonProfilesByProject(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	otherProj := testutil.CreateTestProject(t, db, "other-anon")
	ctx := context.Background()

	for _, name := range []string{"zebra", "apple", "mango"} {
		p := &model.AnonProfile{ProjectID: proj.ID, Name: name, Enabled: true}
		require.NoError(t, model.CreateAnonProfile(ctx, db, p))
	}
	// Profile for a different project — should not appear.
	other := &model.AnonProfile{ProjectID: otherProj.ID, Name: "other-profile", Enabled: true}
	require.NoError(t, model.CreateAnonProfile(ctx, db, other))

	profiles, err := model.ListAnonProfilesByProject(ctx, db, proj.ID)
	require.NoError(t, err)
	assert.Len(t, profiles, 3)
	// Ordered by name.
	assert.Equal(t, "apple", profiles[0].Name)
	assert.Equal(t, "mango", profiles[1].Name)
	assert.Equal(t, "zebra", profiles[2].Name)
}

func TestUpdateAnonProfile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	ctx := context.Background()

	p := &model.AnonProfile{
		ProjectID:   proj.ID,
		Name:        "original-name",
		Description: "original desc",
		Enabled:     true,
	}
	require.NoError(t, model.CreateAnonProfile(ctx, db, p))

	newTags, _ := json.Marshal([]string{"Modality"})
	p.Name = "updated-name"
	p.Description = "updated desc"
	p.RetainedTags = newTags
	p.Enabled = false

	require.NoError(t, model.UpdateAnonProfile(ctx, db, p))

	got, err := model.GetAnonProfileByID(ctx, db, p.ID)
	require.NoError(t, err)
	assert.Equal(t, "updated-name", got.Name)
	assert.Equal(t, "updated desc", got.Description)
	assert.False(t, got.Enabled)

	var retainedTags []string
	require.NoError(t, json.Unmarshal(got.RetainedTags, &retainedTags))
	assert.Equal(t, []string{"Modality"}, retainedTags)
}

func TestDeleteAnonProfile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	ctx := context.Background()

	p := &model.AnonProfile{
		ProjectID: proj.ID,
		Name:      "delete-me-anon",
		Enabled:   true,
	}
	require.NoError(t, model.CreateAnonProfile(ctx, db, p))

	require.NoError(t, model.DeleteAnonProfile(ctx, db, p.ID))

	_, err := model.GetAnonProfileByID(ctx, db, p.ID)
	assert.Error(t, err, "deleted profile should not be retrievable")
}

func TestSetAndGetProjectDefaultAnonProfile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	ctx := context.Background()

	p := &model.AnonProfile{
		ProjectID: proj.ID,
		Name:      "default-profile",
		Enabled:   true,
	}
	require.NoError(t, model.CreateAnonProfile(ctx, db, p))

	require.NoError(t, model.SetProjectDefaultAnonProfile(ctx, db, proj.ID, p.ID))

	got, err := model.GetProjectDefaultAnonProfile(ctx, db, proj.Slug)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, p.ID, got.ID)
}

func TestGetProjectDefaultAnonProfile_NoneSet(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.CreateTestProject(t, db, "no-default-anon")

	got, err := model.GetProjectDefaultAnonProfile(context.Background(), db, proj.Slug)
	require.NoError(t, err)
	assert.Nil(t, got, "should return nil when no default profile is set")
}

func TestSetProjectDefaultAnonProfile_Clear(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	ctx := context.Background()

	p := &model.AnonProfile{
		ProjectID: proj.ID,
		Name:      "to-be-cleared",
		Enabled:   true,
	}
	require.NoError(t, model.CreateAnonProfile(ctx, db, p))
	require.NoError(t, model.SetProjectDefaultAnonProfile(ctx, db, proj.ID, p.ID))

	// Verify it's set.
	got, err := model.GetProjectDefaultAnonProfile(ctx, db, proj.Slug)
	require.NoError(t, err)
	require.NotNil(t, got)

	// Clear it.
	require.NoError(t, model.SetProjectDefaultAnonProfile(ctx, db, proj.ID, ""))

	got, err = model.GetProjectDefaultAnonProfile(ctx, db, proj.Slug)
	require.NoError(t, err)
	assert.Nil(t, got, "should return nil after clearing the default profile")
}
