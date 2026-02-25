package model_test

import (
	"context"
	"testing"
	"time"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeDigestSub(t *testing.T, projectID, email, frequency string, enabled bool) *model.DigestSubscription {
	t.Helper()
	return &model.DigestSubscription{
		Email:     email,
		ProjectID: projectID,
		Frequency: frequency,
		Enabled:   enabled,
	}
}

func TestCreateDigestSubscription(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	ctx := context.Background()

	s := makeDigestSub(t, proj.ID, "digest-create@example.com", "weekly", true)
	require.NoError(t, model.CreateDigestSubscription(ctx, db, s))
	assert.NotEmpty(t, s.ID)
	assert.False(t, s.CreatedAt.IsZero())
}

func TestGetDigestSubscriptionByID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	ctx := context.Background()

	s := makeDigestSub(t, proj.ID, "digest-get@example.com", "monthly", true)
	require.NoError(t, model.CreateDigestSubscription(ctx, db, s))

	got, err := model.GetDigestSubscriptionByID(ctx, db, s.ID)
	require.NoError(t, err)
	assert.Equal(t, s.ID, got.ID)
	assert.Equal(t, "digest-get@example.com", got.Email)
	assert.Equal(t, "monthly", got.Frequency)
	assert.True(t, got.Enabled)
	assert.NotEmpty(t, got.ProjectName) // joined from projects table
}

func TestListDigestSubscriptionsByProject(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	ctx := context.Background()

	s1 := makeDigestSub(t, proj.ID, "alpha@example.com", "weekly", true)
	s2 := makeDigestSub(t, proj.ID, "beta@example.com", "monthly", true)
	require.NoError(t, model.CreateDigestSubscription(ctx, db, s1))
	require.NoError(t, model.CreateDigestSubscription(ctx, db, s2))

	subs, err := model.ListDigestSubscriptionsByProject(ctx, db, proj.ID)
	require.NoError(t, err)
	var emails []string
	for _, s := range subs {
		emails = append(emails, s.Email)
	}
	assert.Contains(t, emails, "alpha@example.com")
	assert.Contains(t, emails, "beta@example.com")
	// Ordered by email: alpha < beta.
	if len(subs) >= 2 {
		assert.Equal(t, "alpha@example.com", subs[len(subs)-2].Email)
	}
}

func TestListAllDigestSubscriptions(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj1 := testutil.SeedProject(t, db)
	proj2 := testutil.CreateTestProject(t, db, "digest-list-all-p2")
	ctx := context.Background()

	s1 := makeDigestSub(t, proj1.ID, "all-proj1@example.com", "weekly", true)
	s2 := makeDigestSub(t, proj2.ID, "all-proj2@example.com", "monthly", true)
	require.NoError(t, model.CreateDigestSubscription(ctx, db, s1))
	require.NoError(t, model.CreateDigestSubscription(ctx, db, s2))

	all, err := model.ListAllDigestSubscriptions(ctx, db)
	require.NoError(t, err)
	var ids []string
	for _, s := range all {
		ids = append(ids, s.ID)
	}
	assert.Contains(t, ids, s1.ID)
	assert.Contains(t, ids, s2.ID)
}

func TestListEnabledDigestSubscriptions(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	ctx := context.Background()

	enabled := makeDigestSub(t, proj.ID, "enabled-only@example.com", "weekly", true)
	disabled := makeDigestSub(t, proj.ID, "disabled-only@example.com", "weekly", false)
	require.NoError(t, model.CreateDigestSubscription(ctx, db, enabled))
	require.NoError(t, model.CreateDigestSubscription(ctx, db, disabled))

	subs, err := model.ListEnabledDigestSubscriptions(ctx, db)
	require.NoError(t, err)
	var ids []string
	for _, s := range subs {
		ids = append(ids, s.ID)
	}
	assert.Contains(t, ids, enabled.ID)
	assert.NotContains(t, ids, disabled.ID)
}

func TestDeleteDigestSubscription(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	ctx := context.Background()

	s := makeDigestSub(t, proj.ID, "digest-delete@example.com", "weekly", true)
	require.NoError(t, model.CreateDigestSubscription(ctx, db, s))
	require.NoError(t, model.DeleteDigestSubscription(ctx, db, s.ID))

	_, err := model.GetDigestSubscriptionByID(ctx, db, s.ID)
	assert.Error(t, err) // not found
}

func TestListDueSubscriptions_NeverSent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	ctx := context.Background()

	// A subscription with no last_sent_at is always due.
	s := makeDigestSub(t, proj.ID, "due-never@example.com", "weekly", true)
	require.NoError(t, model.CreateDigestSubscription(ctx, db, s))

	due, err := model.ListDueSubscriptions(ctx, db)
	require.NoError(t, err)
	var ids []string
	for _, d := range due {
		ids = append(ids, d.ID)
	}
	assert.Contains(t, ids, s.ID)
}

func TestUpdateDigestLastSent_SuppressesDue(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	ctx := context.Background()

	s := makeDigestSub(t, proj.ID, "due-sent@example.com", "weekly", true)
	require.NoError(t, model.CreateDigestSubscription(ctx, db, s))

	// Confirm it's due before sending.
	due, err := model.ListDueSubscriptions(ctx, db)
	require.NoError(t, err)
	var ids []string
	for _, d := range due {
		ids = append(ids, d.ID)
	}
	require.Contains(t, ids, s.ID)

	// Mark as sent — last_sent_at = now().
	require.NoError(t, model.UpdateDigestLastSent(ctx, db, s.ID))

	// Should no longer be due (last_sent_at < 7 days ago is false).
	due2, err := model.ListDueSubscriptions(ctx, db)
	require.NoError(t, err)
	var ids2 []string
	for _, d := range due2 {
		ids2 = append(ids2, d.ID)
	}
	assert.NotContains(t, ids2, s.ID)
}

func TestGetDigestStats(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	ctx := context.Background()

	// Create one study (received).
	testutil.CreateTestStudy(t, db, proj.ID)

	since := time.Now().Add(-1 * time.Hour)
	stats, err := model.GetDigestStats(ctx, db, proj.ID, since)
	require.NoError(t, err)
	assert.Equal(t, proj.ID, stats.ProjectID)
	assert.GreaterOrEqual(t, stats.Received, 1)
	assert.GreaterOrEqual(t, stats.Pending, 1) // received is non-terminal
}
