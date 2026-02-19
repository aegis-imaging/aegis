package model_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/msenjem/aegis/api/model"
	"github.com/msenjem/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateAdminUser(t *testing.T) {
	db := testutil.TestDB(t)

	u := testutil.CreateTestAdminUser(t, db, "admin@test.com", "admin")
	assert.NotEmpty(t, u.ID)
	assert.Equal(t, "admin@test.com", u.Email)
	assert.Equal(t, "admin", u.Role)
	assert.True(t, u.Enabled)
}

func TestGetAdminUserByEmail(t *testing.T) {
	db := testutil.TestDB(t)
	testutil.CreateTestAdminUser(t, db, "find@test.com", "admin")

	found, err := model.GetAdminUserByEmail(context.Background(), db, "find@test.com")
	require.NoError(t, err)
	assert.Equal(t, "find@test.com", found.Email)
}

func TestGetAdminUserByEmail_CaseInsensitive(t *testing.T) {
	db := testutil.TestDB(t)
	testutil.CreateTestAdminUser(t, db, "case@test.com", "admin")

	found, err := model.GetAdminUserByEmail(context.Background(), db, "CASE@TEST.COM")
	require.NoError(t, err)
	assert.Equal(t, "case@test.com", found.Email)
}

func TestGetAdminUserByEmail_NotFound(t *testing.T) {
	db := testutil.TestDB(t)

	_, err := model.GetAdminUserByEmail(context.Background(), db, "nobody@test.com")
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

func TestUpdateAdminUser(t *testing.T) {
	db := testutil.TestDB(t)
	u := testutil.CreateTestAdminUser(t, db, "update@test.com", "admin")

	u.Role = "viewer"
	u.Enabled = false
	err := model.UpdateAdminUser(context.Background(), db, u)
	require.NoError(t, err)

	updated, _ := model.GetAdminUserByEmail(context.Background(), db, "update@test.com")
	assert.Equal(t, "viewer", updated.Role)
	assert.False(t, updated.Enabled)
}

func TestDeleteAdminUser(t *testing.T) {
	db := testutil.TestDB(t)
	u := testutil.CreateTestAdminUser(t, db, "delete@test.com", "admin")

	err := model.DeleteAdminUser(context.Background(), db, u.ID)
	require.NoError(t, err)

	_, err = model.GetAdminUserByEmail(context.Background(), db, "delete@test.com")
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

func TestListAdminUsers(t *testing.T) {
	db := testutil.TestDB(t)
	testutil.CreateTestAdminUser(t, db, "a@test.com", "admin")
	testutil.CreateTestAdminUser(t, db, "b@test.com", "viewer")

	users, err := model.ListAdminUsers(context.Background(), db)
	require.NoError(t, err)
	assert.Len(t, users, 2)
	// Ordered by email
	assert.Equal(t, "a@test.com", users[0].Email)
	assert.Equal(t, "b@test.com", users[1].Email)
}
