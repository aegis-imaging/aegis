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

func TestGenerateEnrollmentTokenUnique(t *testing.T) {
	raw1, hash1, err := model.GenerateEnrollmentToken()
	require.NoError(t, err)
	raw2, hash2, err := model.GenerateEnrollmentToken()
	require.NoError(t, err)
	assert.NotEqual(t, raw1, raw2)
	assert.NotEqual(t, hash1, hash2)
	assert.Len(t, hash1, 64)
	assert.Equal(t, hash1, model.HashEnrollmentToken(raw1))
}

func TestRedeemEnrollmentToken_HappyPath(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	inst := testutil.CreateTestInstitution(t, db, "enroll-happy")

	raw, hash, err := model.GenerateEnrollmentToken()
	require.NoError(t, err)
	tok := &model.SpokeEnrollmentToken{
		InstitutionID: inst.ID,
		Label:         "first",
		CreatedBy:     "admin@test.com",
		ExpiresAt:     time.Now().UTC().Add(1 * time.Hour),
	}
	require.NoError(t, model.CreateSpokeEnrollmentToken(context.Background(), db, tok, hash))
	assert.NotEmpty(t, tok.ID)

	redeemed, err := model.RedeemSpokeEnrollmentToken(context.Background(), db, raw)
	require.NoError(t, err)
	assert.Equal(t, inst.ID, redeemed.InstitutionID)
	require.NotNil(t, redeemed.UsedAt)
}

func TestRedeemEnrollmentToken_DoubleRedemptionRejected(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	inst := testutil.CreateTestInstitution(t, db, "enroll-double")
	raw, hash, _ := model.GenerateEnrollmentToken()
	tok := &model.SpokeEnrollmentToken{
		InstitutionID: inst.ID, ExpiresAt: time.Now().UTC().Add(1 * time.Hour),
	}
	require.NoError(t, model.CreateSpokeEnrollmentToken(context.Background(), db, tok, hash))

	_, err := model.RedeemSpokeEnrollmentToken(context.Background(), db, raw)
	require.NoError(t, err)
	_, err = model.RedeemSpokeEnrollmentToken(context.Background(), db, raw)
	assert.ErrorIs(t, err, model.ErrEnrollmentTokenInvalid)
}

func TestRedeemEnrollmentToken_ExpiredRejected(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	inst := testutil.CreateTestInstitution(t, db, "enroll-expired")
	raw, hash, _ := model.GenerateEnrollmentToken()
	tok := &model.SpokeEnrollmentToken{
		InstitutionID: inst.ID, ExpiresAt: time.Now().UTC().Add(-1 * time.Hour),
	}
	require.NoError(t, model.CreateSpokeEnrollmentToken(context.Background(), db, tok, hash))
	_, err := model.RedeemSpokeEnrollmentToken(context.Background(), db, raw)
	assert.ErrorIs(t, err, model.ErrEnrollmentTokenInvalid)
}

func TestRedeemEnrollmentToken_UnknownRejected(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	_, err := model.RedeemSpokeEnrollmentToken(context.Background(), db, "never-issued")
	assert.ErrorIs(t, err, model.ErrEnrollmentTokenInvalid)
}

func TestRevokeEnrollmentTokenBlocksRedemption(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	inst := testutil.CreateTestInstitution(t, db, "enroll-revoke")
	raw, hash, _ := model.GenerateEnrollmentToken()
	tok := &model.SpokeEnrollmentToken{
		InstitutionID: inst.ID, ExpiresAt: time.Now().UTC().Add(1 * time.Hour),
	}
	require.NoError(t, model.CreateSpokeEnrollmentToken(context.Background(), db, tok, hash))
	require.NoError(t, model.RevokeSpokeEnrollmentToken(context.Background(), db, tok.ID))

	_, err := model.RedeemSpokeEnrollmentToken(context.Background(), db, raw)
	assert.ErrorIs(t, err, model.ErrEnrollmentTokenInvalid)
}

func TestListEnrollmentTokens(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	inst := testutil.CreateTestInstitution(t, db, "enroll-list")
	for i := 0; i < 3; i++ {
		_, hash, _ := model.GenerateEnrollmentToken()
		tok := &model.SpokeEnrollmentToken{
			InstitutionID: inst.ID, ExpiresAt: time.Now().UTC().Add(1 * time.Hour),
		}
		require.NoError(t, model.CreateSpokeEnrollmentToken(context.Background(), db, tok, hash))
	}
	tokens, err := model.ListSpokeEnrollmentTokens(context.Background(), db, inst.ID)
	require.NoError(t, err)
	assert.Len(t, tokens, 3)
}
