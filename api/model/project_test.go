package model_test

import (
	"context"
	"testing"

	"github.com/msenjem/aegis/api/model"
	"github.com/msenjem/aegis/api/testutil"
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
