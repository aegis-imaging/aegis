package model_test

import (
	"context"
	"testing"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListProjects(t *testing.T) {
	db := testutil.TestDB(t)

	projects, err := model.ListProjects(context.Background(), db)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(projects), 1, "default project should exist from migration")
	assert.Equal(t, "default", projects[0].Slug)
}

func TestCreateProject(t *testing.T) {
	db := testutil.TestDB(t)

	p, err := model.CreateProject(context.Background(), db, "Test Project", "test-project", "A test project")
	require.NoError(t, err)
	assert.NotEmpty(t, p.ID)
	assert.Equal(t, "Test Project", p.Name)
	assert.Equal(t, "test-project", p.Slug)
	assert.Equal(t, "A test project", p.Description)
	assert.False(t, p.CreatedAt.IsZero())
}

func TestCreateProject_DuplicateSlug(t *testing.T) {
	db := testutil.TestDB(t)

	_, err := model.CreateProject(context.Background(), db, "Dup", "default", "duplicate of default")
	assert.Error(t, err, "duplicate slug should fail")
}

func TestGetProjectBySlug(t *testing.T) {
	db := testutil.TestDB(t)

	p, err := model.GetProjectBySlug(context.Background(), db, "default")
	require.NoError(t, err)
	assert.Equal(t, "default", p.Slug)
	assert.NotEmpty(t, p.ID)
}

func TestGetProjectBySlug_CaseInsensitive(t *testing.T) {
	db := testutil.TestDB(t)
	_, err := model.CreateProject(context.Background(), db, "Cohort A", "cohort-a", "")
	require.NoError(t, err)

	for _, query := range []string{"cohort-a", "Cohort-A", "COHORT-A", "Cohort-a"} {
		p, err := model.GetProjectBySlug(context.Background(), db, query)
		require.NoError(t, err, "lookup %q should succeed", query)
		assert.Equal(t, "cohort-a", p.Slug, "lookup %q should resolve to canonical slug", query)
	}
}

func TestGetProjectByID(t *testing.T) {
	db := testutil.TestDB(t)
	created, err := model.CreateProject(context.Background(), db, "Find Me", "find-me", "")
	require.NoError(t, err)

	found, err := model.GetProjectByID(context.Background(), db, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, found.ID)
	assert.Equal(t, "Find Me", found.Name)
}

func TestUpdateProject(t *testing.T) {
	db := testutil.TestDB(t)
	created, err := model.CreateProject(context.Background(), db, "Original", "original", "desc")
	require.NoError(t, err)

	updated, err := model.UpdateProject(context.Background(), db, created.ID, "Updated", "updated-slug", "new desc")
	require.NoError(t, err)
	assert.Equal(t, "Updated", updated.Name)
	assert.Equal(t, "updated-slug", updated.Slug)
	assert.Equal(t, "new desc", updated.Description)
}

func TestProjectsWithRetentionPolicy(t *testing.T) {
	db := testutil.TestDB(t)
	ctx := context.Background()

	// Default project from migration has no retention policy.
	// Create one with and one without.
	withPolicy, err := model.CreateProject(ctx, db, "Has Policy", "has-policy", "")
	require.NoError(t, err)
	noPolicy, _ := model.CreateProject(ctx, db, "No Policy", "no-policy", "")
	_ = noPolicy

	days := 30
	require.NoError(t, model.UpdateProjectRetentionDays(ctx, db, withPolicy.ID, &days))

	projects, err := model.ProjectsWithRetentionPolicy(ctx, db)
	require.NoError(t, err)

	found := false
	for _, p := range projects {
		if p.ID == withPolicy.ID {
			found = true
			require.NotNil(t, p.RetentionDays)
			assert.Equal(t, 30, *p.RetentionDays)
		}
		assert.NotEqual(t, noPolicy.ID, p.ID, "project without retention_days should not appear")
	}
	assert.True(t, found, "project with retention_days should appear in results")
}

func TestExpireStudiesByRetention_ExpiresOldApproved(t *testing.T) {
	db := testutil.TestDB(t)
	ctx := context.Background()

	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	// Promote to approved and backdate created_at to 100 days ago.
	require.NoError(t, model.UpdateStudyStatus(ctx, db, study.ID, "approved"))
	_, err := db.ExecContext(ctx, `UPDATE studies SET created_at = now() - INTERVAL '100 days' WHERE id = $1`, study.ID)
	require.NoError(t, err)

	n, err := model.ExpireStudiesByRetention(ctx, db, proj.ID, 30)
	require.NoError(t, err)
	assert.EqualValues(t, 1, n, "one old approved study should be expired")

	updated, err := model.GetStudyByID(ctx, db, study.ID)
	require.NoError(t, err)
	assert.Equal(t, "expired", updated.Status)
}

func TestExpireStudiesByRetention_PreservesRecentStudies(t *testing.T) {
	db := testutil.TestDB(t)
	ctx := context.Background()

	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)
	require.NoError(t, model.UpdateStudyStatus(ctx, db, study.ID, "approved"))
	// created_at is now() — well within a 30-day window.

	n, err := model.ExpireStudiesByRetention(ctx, db, proj.ID, 30)
	require.NoError(t, err)
	assert.EqualValues(t, 0, n, "recent approved study should not be expired")

	updated, err := model.GetStudyByID(ctx, db, study.ID)
	require.NoError(t, err)
	assert.Equal(t, "approved", updated.Status)
}

func TestExpireStudiesByRetention_OnlyExpireApproved(t *testing.T) {
	db := testutil.TestDB(t)
	ctx := context.Background()

	proj := testutil.SeedProject(t, db)

	// Create old studies in non-approved statuses.
	for _, status := range []string{"received", "rejected", "defacing"} {
		s := testutil.CreateTestStudy(t, db, proj.ID)
		require.NoError(t, model.UpdateStudyStatus(ctx, db, s.ID, status))
		_, err := db.ExecContext(ctx, `UPDATE studies SET created_at = now() - INTERVAL '100 days' WHERE id = $1`, s.ID)
		require.NoError(t, err)
	}

	n, err := model.ExpireStudiesByRetention(ctx, db, proj.ID, 30)
	require.NoError(t, err)
	assert.EqualValues(t, 0, n, "non-approved studies should never be expired by retention policy")
}
