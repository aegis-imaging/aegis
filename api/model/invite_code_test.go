package model_test

import (
	"context"
	"database/sql"
	"regexp"
	"testing"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var inviteCodePattern = regexp.MustCompile(`^[A-Z0-9]{4}-[A-Z0-9]{4}-[A-Z0-9]{4}$`)

func TestCreateInviteCode(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)

	ic, err := model.CreateInviteCode(context.Background(), db, "Dr. Smith – Stanford")
	require.NoError(t, err)
	assert.NotEmpty(t, ic.ID)
	assert.True(t, ic.Enabled)
	assert.Equal(t, "Dr. Smith – Stanford", ic.Label)
	assert.True(t, inviteCodePattern.MatchString(ic.Code),
		"code %q should match XXXX-XXXX-XXXX format", ic.Code)
	assert.Nil(t, ic.UsedAt)
	assert.Nil(t, ic.UsedByIP)
}

func TestGetInviteCode(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)

	created, err := model.CreateInviteCode(context.Background(), db, "test-get")
	require.NoError(t, err)

	got, err := model.GetInviteCode(context.Background(), db, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, got.ID)
	assert.Equal(t, created.Code, got.Code)
	assert.True(t, got.Enabled)
}

func TestListInviteCodes(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	ctx := context.Background()

	_, err := model.CreateInviteCode(ctx, db, "first")
	require.NoError(t, err)
	_, err = model.CreateInviteCode(ctx, db, "second")
	require.NoError(t, err)

	codes, err := model.ListInviteCodes(ctx, db)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(codes), 2)
	// Newest first.
	assert.Equal(t, "second", codes[0].Label)
}

func TestValidateAndRecordInviteCode_Valid(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	ctx := context.Background()

	ic, err := model.CreateInviteCode(ctx, db, "valid-test")
	require.NoError(t, err)

	ok, err := model.ValidateAndRecordInviteCode(ctx, db, ic.Code, "1.1.1.1")
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestValidateAndRecordInviteCode_Invalid(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)

	ok, err := model.ValidateAndRecordInviteCode(context.Background(), db, "ZZZZ-ZZZZ-ZZZZ", "1.1.1.1")
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestValidateAndRecordInviteCode_Disabled(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	ctx := context.Background()

	ic, err := model.CreateInviteCode(ctx, db, "disabled-test")
	require.NoError(t, err)

	require.NoError(t, model.RevokeInviteCode(ctx, db, ic.ID))

	ok, err := model.ValidateAndRecordInviteCode(ctx, db, ic.Code, "2.2.2.2")
	require.NoError(t, err)
	assert.False(t, ok, "disabled code should not validate")
}

func TestValidateAndRecordInviteCode_FirstUseRecorded(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	ctx := context.Background()

	ic, err := model.CreateInviteCode(ctx, db, "first-use")
	require.NoError(t, err)
	assert.Nil(t, ic.UsedAt)

	ok, err := model.ValidateAndRecordInviteCode(ctx, db, ic.Code, "3.3.3.3")
	require.NoError(t, err)
	require.True(t, ok)

	got, err := model.GetInviteCode(ctx, db, ic.ID)
	require.NoError(t, err)
	assert.NotNil(t, got.UsedAt, "used_at should be recorded on first use")
	require.NotNil(t, got.UsedByIP)
	assert.Equal(t, "3.3.3.3", *got.UsedByIP)
}

func TestValidateAndRecordInviteCode_SecondUsePreservesFirstIP(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	ctx := context.Background()

	ic, err := model.CreateInviteCode(ctx, db, "second-use")
	require.NoError(t, err)

	_, err = model.ValidateAndRecordInviteCode(ctx, db, ic.Code, "4.4.4.4")
	require.NoError(t, err)
	_, err = model.ValidateAndRecordInviteCode(ctx, db, ic.Code, "5.5.5.5")
	require.NoError(t, err)

	got, err := model.GetInviteCode(ctx, db, ic.ID)
	require.NoError(t, err)
	require.NotNil(t, got.UsedByIP)
	// COALESCE preserves first IP.
	assert.Equal(t, "4.4.4.4", *got.UsedByIP, "first IP should be preserved on repeated use")
}

func TestRevokeInviteCode(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	ctx := context.Background()

	ic, err := model.CreateInviteCode(ctx, db, "revoke-me")
	require.NoError(t, err)
	assert.True(t, ic.Enabled)

	require.NoError(t, model.RevokeInviteCode(ctx, db, ic.ID))

	got, err := model.GetInviteCode(ctx, db, ic.ID)
	require.NoError(t, err)
	assert.False(t, got.Enabled)
}

func TestRevokeInviteCode_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)

	err := model.RevokeInviteCode(context.Background(), db, "00000000-0000-0000-0000-000000000000")
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

func TestDeleteInviteCode(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	ctx := context.Background()

	ic, err := model.CreateInviteCode(ctx, db, "delete-me")
	require.NoError(t, err)

	require.NoError(t, model.DeleteInviteCode(ctx, db, ic.ID))

	_, err = model.GetInviteCode(ctx, db, ic.ID)
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

func TestDeleteInviteCode_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)

	err := model.DeleteInviteCode(context.Background(), db, "00000000-0000-0000-0000-000000000000")
	assert.ErrorIs(t, err, sql.ErrNoRows)
}
