package model_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func shareTokenHash(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

func TestCreateExportShare(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	tok := fmt.Sprintf("tok_%d", time.Now().UnixNano())
	exp := time.Now().UTC().Add(48 * time.Hour)

	s, err := model.CreateExportShare(context.Background(), db, study.ID, shareTokenHash(tok),
		"recipient@example.com", "test note", "admin@test.com", exp, nil)
	require.NoError(t, err)
	assert.NotEmpty(t, s.ID)
	assert.Equal(t, study.ID, s.StudyID)
	assert.Equal(t, "recipient@example.com", s.RecipientEmail)
	assert.Equal(t, "test note", s.Note)
	assert.Equal(t, "admin@test.com", s.CreatedBy)
	assert.Nil(t, s.RevokedAt)
	assert.Nil(t, s.MaxDownloads)
}

func TestCreateExportShare_WithMaxDownloads(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	tok := fmt.Sprintf("tok_max_%d", time.Now().UnixNano())
	exp := time.Now().UTC().Add(24 * time.Hour)
	maxDL := 3

	s, err := model.CreateExportShare(context.Background(), db, study.ID, shareTokenHash(tok),
		"r@example.com", "", "admin@test.com", exp, &maxDL)
	require.NoError(t, err)
	require.NotNil(t, s.MaxDownloads)
	assert.Equal(t, 3, *s.MaxDownloads)
}

func TestGetExportShareByTokenHash(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	tok := fmt.Sprintf("tok_get_%d", time.Now().UnixNano())
	exp := time.Now().UTC().Add(24 * time.Hour)
	created, err := model.CreateExportShare(context.Background(), db, study.ID, shareTokenHash(tok),
		"get@example.com", "", "admin@test.com", exp, nil)
	require.NoError(t, err)

	got, err := model.GetExportShareByTokenHash(context.Background(), db, shareTokenHash(tok))
	require.NoError(t, err)
	assert.Equal(t, created.ID, got.ID)
	assert.Equal(t, "get@example.com", got.RecipientEmail)
	assert.Equal(t, 0, got.DownloadCount)
}

func TestGetExportShareByID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	tok := fmt.Sprintf("tok_byid_%d", time.Now().UnixNano())
	exp := time.Now().UTC().Add(24 * time.Hour)
	created, err := model.CreateExportShare(context.Background(), db, study.ID, shareTokenHash(tok),
		"byid@example.com", "", "admin@test.com", exp, nil)
	require.NoError(t, err)

	got, err := model.GetExportShareByID(context.Background(), db, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, got.ID)
}

func TestListExportSharesByStudy(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)
	other := testutil.CreateTestStudy(t, db, proj.ID)

	exp := time.Now().UTC().Add(24 * time.Hour)

	tok1 := fmt.Sprintf("tok_list_1_%d", time.Now().UnixNano())
	tok2 := fmt.Sprintf("tok_list_2_%d", time.Now().UnixNano()+1)
	tokOther := fmt.Sprintf("tok_list_other_%d", time.Now().UnixNano()+2)

	ctx := context.Background()
	_, err := model.CreateExportShare(ctx, db, study.ID, shareTokenHash(tok1), "a@b.com", "", "me", exp, nil)
	require.NoError(t, err)
	_, err = model.CreateExportShare(ctx, db, study.ID, shareTokenHash(tok2), "c@d.com", "", "me", exp, nil)
	require.NoError(t, err)
	_, err = model.CreateExportShare(ctx, db, other.ID, shareTokenHash(tokOther), "e@f.com", "", "me", exp, nil)
	require.NoError(t, err)

	shares, err := model.ListExportSharesByStudy(ctx, db, study.ID)
	require.NoError(t, err)
	assert.Len(t, shares, 2, "only shares for the target study should be returned")
}

func TestRevokeExportShare(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)
	ctx := context.Background()

	tok := fmt.Sprintf("tok_revoke_%d", time.Now().UnixNano())
	exp := time.Now().UTC().Add(24 * time.Hour)
	s, err := model.CreateExportShare(ctx, db, study.ID, shareTokenHash(tok), "r@r.com", "", "me", exp, nil)
	require.NoError(t, err)
	assert.Nil(t, s.RevokedAt)

	require.NoError(t, model.RevokeExportShare(ctx, db, s.ID, ""))

	got, err := model.GetExportShareByID(ctx, db, s.ID)
	require.NoError(t, err)
	assert.NotNil(t, got.RevokedAt)
	assert.Nil(t, got.RevocationReason)
}

func TestRevokeExportShare_WithReason(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)
	ctx := context.Background()

	tok := fmt.Sprintf("tok_revoke_reason_%d", time.Now().UnixNano())
	exp := time.Now().UTC().Add(24 * time.Hour)
	s, err := model.CreateExportShare(ctx, db, study.ID, shareTokenHash(tok), "rr@rr.com", "", "me", exp, nil)
	require.NoError(t, err)

	require.NoError(t, model.RevokeExportShare(ctx, db, s.ID, "data embargo"))

	got, err := model.GetExportShareByID(ctx, db, s.ID)
	require.NoError(t, err)
	require.NotNil(t, got.RevocationReason)
	assert.Equal(t, "data embargo", *got.RevocationReason)
}

func TestExtendExportShare(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)
	ctx := context.Background()

	tok := fmt.Sprintf("tok_extend_%d", time.Now().UnixNano())
	exp := time.Now().UTC().Add(24 * time.Hour)
	s, err := model.CreateExportShare(ctx, db, study.ID, shareTokenHash(tok), "ext@ext.com", "", "me", exp, nil)
	require.NoError(t, err)

	newExp := time.Now().UTC().Add(72 * time.Hour)
	require.NoError(t, model.ExtendExportShare(ctx, db, s.ID, newExp))

	got, err := model.GetExportShareByID(ctx, db, s.ID)
	require.NoError(t, err)
	assert.WithinDuration(t, newExp, got.ExpiresAt, time.Second)
}

func TestExtendExportShare_Revoked_ReturnsError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)
	ctx := context.Background()

	tok := fmt.Sprintf("tok_extend_rev_%d", time.Now().UnixNano())
	exp := time.Now().UTC().Add(24 * time.Hour)
	s, err := model.CreateExportShare(ctx, db, study.ID, shareTokenHash(tok), "ev@ev.com", "", "me", exp, nil)
	require.NoError(t, err)

	require.NoError(t, model.RevokeExportShare(ctx, db, s.ID, ""))

	err = model.ExtendExportShare(ctx, db, s.ID, time.Now().UTC().Add(48*time.Hour))
	assert.Error(t, err, "extending a revoked share should return an error")
}

func TestCreateExportDownload_AndCount(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)
	ctx := context.Background()

	tok := fmt.Sprintf("tok_dl_%d", time.Now().UnixNano())
	exp := time.Now().UTC().Add(24 * time.Hour)
	s, err := model.CreateExportShare(ctx, db, study.ID, shareTokenHash(tok), "dl@dl.com", "", "me", exp, nil)
	require.NoError(t, err)

	require.NoError(t, model.CreateExportDownload(ctx, db, s.ID, "1.2.3.4"))
	require.NoError(t, model.CreateExportDownload(ctx, db, s.ID, "1.2.3.5"))

	n, err := model.GetShareDownloadCount(ctx, db, s.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, n)
}

func TestListExportDownloadsByShare(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)
	ctx := context.Background()

	tok := fmt.Sprintf("tok_dllist_%d", time.Now().UnixNano())
	exp := time.Now().UTC().Add(24 * time.Hour)
	s, err := model.CreateExportShare(ctx, db, study.ID, shareTokenHash(tok), "lst@lst.com", "", "me", exp, nil)
	require.NoError(t, err)

	require.NoError(t, model.CreateExportDownload(ctx, db, s.ID, "10.0.0.1"))
	require.NoError(t, model.CreateExportDownload(ctx, db, s.ID, "10.0.0.2"))
	require.NoError(t, model.CreateExportDownload(ctx, db, s.ID, "10.0.0.3"))

	downloads, err := model.ListExportDownloadsByShare(ctx, db, s.ID)
	require.NoError(t, err)
	assert.Len(t, downloads, 3)
	assert.Equal(t, s.ID, downloads[0].ShareID)
}

func TestGetExportDownloadAnalytics_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)

	analytics, err := model.GetExportDownloadAnalytics(context.Background(), db)
	require.NoError(t, err)
	assert.NotNil(t, analytics)
	assert.NotNil(t, analytics.Last30Days)
	assert.NotNil(t, analytics.TopShares)
}

func TestListAllExportShares_StatusFilter(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)
	ctx := context.Background()

	future := time.Now().UTC().Add(48 * time.Hour)
	past := time.Now().UTC().Add(-24 * time.Hour)

	tokActive := fmt.Sprintf("tok_active_%d", time.Now().UnixNano())
	tokExpired := fmt.Sprintf("tok_expired_%d", time.Now().UnixNano()+1)
	tokRevoked := fmt.Sprintf("tok_revoked_%d", time.Now().UnixNano()+2)

	active, err := model.CreateExportShare(ctx, db, study.ID, shareTokenHash(tokActive), "active@s.com", "", "me", future, nil)
	require.NoError(t, err)
	_, err = model.CreateExportShare(ctx, db, study.ID, shareTokenHash(tokExpired), "expired@s.com", "", "me", past, nil)
	require.NoError(t, err)
	revoked, err := model.CreateExportShare(ctx, db, study.ID, shareTokenHash(tokRevoked), "revoked@s.com", "", "me", future, nil)
	require.NoError(t, err)

	require.NoError(t, model.RevokeExportShare(ctx, db, revoked.ID, ""))

	nActive, err := model.CountAllExportShares(ctx, db, model.ShareStatusActive)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, nActive, 1)

	nRevoked, err := model.CountAllExportShares(ctx, db, model.ShareStatusRevoked)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, nRevoked, 1)

	nExpired, err := model.CountAllExportShares(ctx, db, model.ShareStatusExpired)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, nExpired, 1)

	nAll, err := model.CountAllExportShares(ctx, db, "")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, nAll, 3)

	// The active share should appear in the ListAllExportShares active filter.
	activeShares, err := model.ListAllExportShares(ctx, db, model.ShareStatusActive, 100, 0)
	require.NoError(t, err)
	ids := make([]string, len(activeShares))
	for i, s := range activeShares {
		ids[i] = s.ID
	}
	assert.Contains(t, ids, active.ID)
}
