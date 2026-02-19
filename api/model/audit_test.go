package model_test

import (
	"context"
	"testing"

	"github.com/msenjem/aegis/api/model"
	"github.com/msenjem/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateAuditEntry(t *testing.T) {
	db := testutil.TestDB(t)

	err := model.CreateAuditEntry(context.Background(), db,
		"study.approved", "admin@test.com", "study", "study-123", "127.0.0.1",
		map[string]any{"note": "test"})
	require.NoError(t, err)
}

func TestCreateAuditEntry_NilDetail(t *testing.T) {
	db := testutil.TestDB(t)

	err := model.CreateAuditEntry(context.Background(), db,
		"study.rejected", "admin@test.com", "study", "study-456", "", nil)
	require.NoError(t, err)
}

func TestListAuditEntries(t *testing.T) {
	db := testutil.TestDB(t)

	model.CreateAuditEntry(context.Background(), db, "study.approved", "a", "study", "s1", "", nil)
	model.CreateAuditEntry(context.Background(), db, "study.rejected", "a", "study", "s2", "", nil)
	model.CreateAuditEntry(context.Background(), db, "user.created", "a", "user", "u1", "", nil)

	entries, err := model.ListAuditEntries(context.Background(), db, "", "", 0)
	require.NoError(t, err)
	assert.Len(t, entries, 3)
}

func TestListAuditEntries_FilterByAction(t *testing.T) {
	db := testutil.TestDB(t)

	model.CreateAuditEntry(context.Background(), db, "study.approved", "a", "study", "s1", "", nil)
	model.CreateAuditEntry(context.Background(), db, "study.rejected", "a", "study", "s2", "", nil)

	entries, err := model.ListAuditEntries(context.Background(), db, "study.approved", "", 0)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "study.approved", entries[0].Action)
}

func TestListAuditEntries_FilterByResourceType(t *testing.T) {
	db := testutil.TestDB(t)

	model.CreateAuditEntry(context.Background(), db, "study.approved", "a", "study", "s1", "", nil)
	model.CreateAuditEntry(context.Background(), db, "user.created", "a", "user", "u1", "", nil)

	entries, err := model.ListAuditEntries(context.Background(), db, "", "user", 0)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "user", entries[0].ResourceType)
}

func TestListAuditEntries_WithLimit(t *testing.T) {
	db := testutil.TestDB(t)

	for i := 0; i < 5; i++ {
		model.CreateAuditEntry(context.Background(), db, "test.event", "a", "test", "t", "", nil)
	}

	entries, err := model.ListAuditEntries(context.Background(), db, "", "", 2)
	require.NoError(t, err)
	assert.Len(t, entries, 2)
}
