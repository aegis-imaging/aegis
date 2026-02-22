package model_test

import (
	"context"
	"testing"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
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

	entries, err := model.ListAuditEntries(context.Background(), db, model.AuditFilters{}, 0, 0)
	require.NoError(t, err)
	assert.Len(t, entries, 3)
}

func TestListAuditEntries_FilterByAction(t *testing.T) {
	db := testutil.TestDB(t)

	model.CreateAuditEntry(context.Background(), db, "study.approved", "a", "study", "s1", "", nil)
	model.CreateAuditEntry(context.Background(), db, "study.rejected", "a", "study", "s2", "", nil)

	// Exact match — action LIKE 'study.approved%'
	entries, err := model.ListAuditEntries(context.Background(), db, model.AuditFilters{Action: "study.approved"}, 0, 0)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "study.approved", entries[0].Action)
}

func TestListAuditEntries_FilterByActionPrefix(t *testing.T) {
	db := testutil.TestDB(t)

	model.CreateAuditEntry(context.Background(), db, "study.approved", "a", "study", "s1", "", nil)
	model.CreateAuditEntry(context.Background(), db, "study.rejected", "a", "study", "s2", "", nil)
	model.CreateAuditEntry(context.Background(), db, "user.created", "a", "user", "u1", "", nil)

	// Prefix match — action LIKE 'study%'
	entries, err := model.ListAuditEntries(context.Background(), db, model.AuditFilters{Action: "study"}, 0, 0)
	require.NoError(t, err)
	assert.Len(t, entries, 2)
}

func TestListAuditEntries_FilterByResourceType(t *testing.T) {
	db := testutil.TestDB(t)

	model.CreateAuditEntry(context.Background(), db, "study.approved", "a", "study", "s1", "", nil)
	model.CreateAuditEntry(context.Background(), db, "user.created", "a", "user", "u1", "", nil)

	entries, err := model.ListAuditEntries(context.Background(), db, model.AuditFilters{ResourceType: "user"}, 0, 0)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "user", entries[0].ResourceType)
}

func TestListAuditEntries_FilterByActor(t *testing.T) {
	db := testutil.TestDB(t)

	model.CreateAuditEntry(context.Background(), db, "study.approved", "alice@test.com", "study", "s1", "", nil)
	model.CreateAuditEntry(context.Background(), db, "study.rejected", "bob@test.com", "study", "s2", "", nil)

	entries, err := model.ListAuditEntries(context.Background(), db, model.AuditFilters{Actor: "alice@test.com"}, 0, 0)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "alice@test.com", entries[0].Actor)
}

func TestListAuditEntries_WithLimit(t *testing.T) {
	db := testutil.TestDB(t)

	for i := 0; i < 5; i++ {
		model.CreateAuditEntry(context.Background(), db, "test.event", "a", "test", "t", "", nil)
	}

	entries, err := model.ListAuditEntries(context.Background(), db, model.AuditFilters{}, 2, 0)
	require.NoError(t, err)
	assert.Len(t, entries, 2)
}

func TestListAuditEntries_Pagination(t *testing.T) {
	db := testutil.TestDB(t)

	for i := 0; i < 5; i++ {
		model.CreateAuditEntry(context.Background(), db, "test.event", "a", "test", "t", "", nil)
	}

	page1, err := model.ListAuditEntries(context.Background(), db, model.AuditFilters{}, 2, 0)
	require.NoError(t, err)
	assert.Len(t, page1, 2)

	page2, err := model.ListAuditEntries(context.Background(), db, model.AuditFilters{}, 2, 2)
	require.NoError(t, err)
	assert.Len(t, page2, 2)

	assert.NotEqual(t, page1[0].ID, page2[0].ID, "pages should not overlap")
}

func TestCountAuditEntries(t *testing.T) {
	db := testutil.TestDB(t)

	model.CreateAuditEntry(context.Background(), db, "study.approved", "a", "study", "s1", "", nil)
	model.CreateAuditEntry(context.Background(), db, "study.rejected", "a", "study", "s2", "", nil)
	model.CreateAuditEntry(context.Background(), db, "user.created", "a", "user", "u1", "", nil)

	total, err := model.CountAuditEntries(context.Background(), db, model.AuditFilters{})
	require.NoError(t, err)
	assert.Equal(t, 3, total)

	studyTotal, err := model.CountAuditEntries(context.Background(), db, model.AuditFilters{Action: "study"})
	require.NoError(t, err)
	assert.Equal(t, 2, studyTotal)
}
