package model_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateInviteRequest(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)

	r, err := model.CreateInviteRequest(context.Background(), db,
		"Alice Smith", "alice@example.com", "University", "Looking forward to it", "1.2.3.4")
	require.NoError(t, err)
	assert.NotEmpty(t, r.ID)
	assert.Equal(t, "Alice Smith", r.Name)
	assert.Equal(t, "alice@example.com", r.Email)
	assert.Equal(t, "University", r.Org)
	assert.Equal(t, "pending", r.Status)
	require.NotNil(t, r.IP)
	assert.Equal(t, "1.2.3.4", *r.IP)
	assert.False(t, r.CreatedAt.IsZero())
}

func TestGetInviteRequest(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	ctx := context.Background()

	r, err := model.CreateInviteRequest(ctx, db, "Bob", "bob@example.com", "Hospital", "", "")
	require.NoError(t, err)

	got, err := model.GetInviteRequest(ctx, db, r.ID)
	require.NoError(t, err)
	assert.Equal(t, r.ID, got.ID)
	assert.Equal(t, "Bob", got.Name)
	assert.Equal(t, "bob@example.com", got.Email)
	assert.Equal(t, "pending", got.Status)
}

func TestGetInviteRequest_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)

	_, err := model.GetInviteRequest(context.Background(), db, "00000000-0000-0000-0000-000000000000")
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

func TestListInviteRequests_All(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	ctx := context.Background()

	r1, err := model.CreateInviteRequest(ctx, db, "Carol", "carol@example.com", "Lab", "", "")
	require.NoError(t, err)
	r2, err := model.CreateInviteRequest(ctx, db, "Dave", "dave@example.com", "Clinic", "", "")
	require.NoError(t, err)

	all, err := model.ListInviteRequests(ctx, db, "")
	require.NoError(t, err)
	var ids []string
	for _, r := range all {
		ids = append(ids, r.ID)
	}
	assert.Contains(t, ids, r1.ID)
	assert.Contains(t, ids, r2.ID)
}

func TestListInviteRequests_FilterByStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	ctx := context.Background()

	pending, err := model.CreateInviteRequest(ctx, db, "Eve", "eve@example.com", "Org", "", "")
	require.NoError(t, err)

	// Deny another request to give it a different status.
	toApprove, err := model.CreateInviteRequest(ctx, db, "Frank", "frank@example.com", "Org2", "", "")
	require.NoError(t, err)
	require.NoError(t, model.DenyInviteRequest(ctx, db, toApprove.ID, "admin@example.com"))

	// Filter to pending only.
	pendingList, err := model.ListInviteRequests(ctx, db, "pending")
	require.NoError(t, err)
	var pendingIDs []string
	for _, r := range pendingList {
		pendingIDs = append(pendingIDs, r.ID)
	}
	assert.Contains(t, pendingIDs, pending.ID)
	assert.NotContains(t, pendingIDs, toApprove.ID)
}

func TestApproveInviteRequest(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	ctx := context.Background()

	// Create an invite code to link.
	ic, err := model.CreateInviteCode(ctx, db, "Test Approver Code")
	require.NoError(t, err)

	r, err := model.CreateInviteRequest(ctx, db, "Grace", "grace@example.com", "Institute", "", "")
	require.NoError(t, err)

	require.NoError(t, model.ApproveInviteRequest(ctx, db, r.ID, "admin@example.com", ic.ID))

	got, err := model.GetInviteRequest(ctx, db, r.ID)
	require.NoError(t, err)
	assert.Equal(t, "approved", got.Status)
	require.NotNil(t, got.ReviewedBy)
	assert.Equal(t, "admin@example.com", *got.ReviewedBy)
	require.NotNil(t, got.InviteCodeID)
	assert.Equal(t, ic.ID, *got.InviteCodeID)
	assert.NotNil(t, got.ReviewedAt)
}

func TestApproveInviteRequest_AlreadyDenied(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	ctx := context.Background()

	r, err := model.CreateInviteRequest(ctx, db, "Heidi", "heidi@example.com", "Corp", "", "")
	require.NoError(t, err)
	require.NoError(t, model.DenyInviteRequest(ctx, db, r.ID, "admin@example.com"))

	// Approving a non-pending request returns ErrNoRows.
	err = model.ApproveInviteRequest(ctx, db, r.ID, "admin@example.com", "00000000-0000-0000-0000-000000000001")
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

func TestDenyInviteRequest(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	ctx := context.Background()

	r, err := model.CreateInviteRequest(ctx, db, "Ivan", "ivan@example.com", "Research", "", "")
	require.NoError(t, err)

	require.NoError(t, model.DenyInviteRequest(ctx, db, r.ID, "admin@example.com"))

	got, err := model.GetInviteRequest(ctx, db, r.ID)
	require.NoError(t, err)
	assert.Equal(t, "denied", got.Status)
	require.NotNil(t, got.ReviewedBy)
	assert.Equal(t, "admin@example.com", *got.ReviewedBy)
}

func TestDenyInviteRequest_AlreadyApproved(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	ctx := context.Background()

	r, err := model.CreateInviteRequest(ctx, db, "Judy", "judy@example.com", "Lab", "", "")
	require.NoError(t, err)

	ic, err := model.CreateInviteCode(ctx, db, "Deny Test Code")
	require.NoError(t, err)
	require.NoError(t, model.ApproveInviteRequest(ctx, db, r.ID, "admin@example.com", ic.ID))

	// Denying an already-approved request returns ErrNoRows.
	err = model.DenyInviteRequest(ctx, db, r.ID, "admin@example.com")
	assert.ErrorIs(t, err, sql.ErrNoRows)
}
