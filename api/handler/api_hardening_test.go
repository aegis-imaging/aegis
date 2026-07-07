package handler_test

// Tests for the API hardening fixes: atomic invite claiming, atomic upload
// session transitions, uploader-login failure responses, bcrypt length cap,
// and the STOW-RS request size cap.

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── Invite claim: concurrent redeems, one winner ─────────────────────────────

func TestClaimUploaderInvite_ConcurrentSingleWinner(t *testing.T) {
	db := testutil.TestDB(t)

	proj := testutil.CreateTestProject(t, db, "ClaimRace")
	inv, err := model.CreateUploaderInvite(context.Background(), db, model.CreateUploaderInviteInput{
		Email:     "race@hospital.test",
		ProjectID: proj.ID,
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
	})
	require.NoError(t, err)

	const racers = 8
	var wins atomic.Int32
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < racers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			claimed, err := model.ClaimUploaderInvite(context.Background(), db, inv.ID)
			assert.NoError(t, err)
			if claimed {
				wins.Add(1)
			}
		}()
	}
	close(start)
	wg.Wait()

	assert.Equal(t, int32(1), wins.Load(), "exactly one concurrent redeem must win the claim")

	// Unclaim makes the invite redeemable again.
	require.NoError(t, model.UnclaimUploaderInvite(context.Background(), db, inv.ID))
	claimed, err := model.ClaimUploaderInvite(context.Background(), db, inv.ID)
	require.NoError(t, err)
	assert.True(t, claimed, "invite must be claimable again after unclaim")
}

func TestRedeemUploaderInvite_RoleConflictUnclaims(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	proj := testutil.CreateTestProject(t, db, "RoleConflict")
	// The invite email already belongs to a real admin — redeem must refuse
	// to hijack the account.
	testutil.CreateTestAdminUser(t, db, "taken@aegis.test", "admin")
	inv, err := model.CreateUploaderInvite(context.Background(), db, model.CreateUploaderInviteInput{
		Email:     "taken@aegis.test",
		ProjectID: proj.ID,
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
	})
	require.NoError(t, err)

	body, _ := json.Marshal(map[string]string{"password": "validpassword99"})
	req := httptest.NewRequest("POST", "/api/uploader-invites/"+inv.InviteToken+"/redeem", bytes.NewReader(body))
	req.SetPathValue("token", inv.InviteToken)
	rr := httptest.NewRecorder()
	srv.RedeemUploaderInvite(rr, req)
	assert.Equal(t, http.StatusConflict, rr.Code, rr.Body.String())

	// Provisioning failed after the claim, so the invite must be un-claimed
	// (still redeemable, e.g. after the admin re-issues to a fresh address).
	fresh, err := model.GetUploaderInviteByToken(context.Background(), db, inv.InviteToken)
	require.NoError(t, err)
	assert.Nil(t, fresh.RedeemedAt, "invite must be un-claimed after provisioning failure")
}

// ─── Uploader login: failure paths stay 401 with an identical body ────────────

func TestUploaderLogin_FailurePathsReturn401(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	// Wrong role: a real admin account must not be able to log in here.
	testutil.CreateTestAdminUser(t, db, "adminrole@aegis.test", "admin")
	// No password hash: uploader account that never finished redemption.
	testutil.CreateTestAdminUser(t, db, "nohash@hospital.test", "uploader")

	cases := []struct {
		name  string
		email string
	}{
		{"unknown email", "ghost@nowhere.test"},
		{"wrong role", "adminrole@aegis.test"},
		{"no password hash", "nohash@hospital.test"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body, _ := json.Marshal(map[string]string{
				"email":    tc.email,
				"password": "someattempt123",
			})
			req := httptest.NewRequest("POST", "/api/auth/uploader-login", bytes.NewReader(body))
			rr := httptest.NewRecorder()
			srv.UploaderLogin(rr, req)
			assert.Equal(t, http.StatusUnauthorized, rr.Code)
			assert.Contains(t, rr.Body.String(), "invalid credentials",
				"all failure paths must return the same generic message")
		})
	}
}

// ─── bcrypt 72-byte cap ────────────────────────────────────────────────────────

func TestRedeemUploaderInvite_RejectsOver72BytePassword(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	proj := testutil.CreateTestProject(t, db, "LongPwd")
	inv, err := model.CreateUploaderInvite(context.Background(), db, model.CreateUploaderInviteInput{
		Email:     "longpwd@hospital.test",
		ProjectID: proj.ID,
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
	})
	require.NoError(t, err)

	body, _ := json.Marshal(map[string]string{"password": strings.Repeat("a", 73)})
	req := httptest.NewRequest("POST", "/api/uploader-invites/"+inv.InviteToken+"/redeem", bytes.NewReader(body))
	req.SetPathValue("token", inv.InviteToken)
	rr := httptest.NewRecorder()
	srv.RedeemUploaderInvite(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code, rr.Body.String())
	assert.Contains(t, rr.Body.String(), "72")

	// Exactly 72 bytes is fine (bcrypt's real limit).
	body, _ = json.Marshal(map[string]string{"password": strings.Repeat("a", 72)})
	req = httptest.NewRequest("POST", "/api/uploader-invites/"+inv.InviteToken+"/redeem", bytes.NewReader(body))
	req.SetPathValue("token", inv.InviteToken)
	rr = httptest.NewRecorder()
	srv.RedeemUploaderInvite(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
}

// ─── Upload session: concurrent completes, one winner ─────────────────────────

func TestMarkUploadSessionIngesting_ConcurrentSingleWinner(t *testing.T) {
	db := testutil.TestDB(t)

	proj := testutil.CreateTestProject(t, db, "SessionRace")
	sess, err := model.CreateUploadSession(context.Background(), db, proj.ID, 1, "uploads/race", "", "")
	require.NoError(t, err)
	require.Equal(t, "initiated", sess.Status)

	const racers = 8
	var wins atomic.Int32
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < racers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			claimed, err := model.MarkUploadSessionIngesting(context.Background(), db, sess.ID)
			assert.NoError(t, err)
			if claimed {
				wins.Add(1)
			}
		}()
	}
	close(start)
	wg.Wait()

	assert.Equal(t, int32(1), wins.Load(), "exactly one concurrent complete must claim the session")
}

// ─── STOW-RS request size cap ──────────────────────────────────────────────────

func TestStowReceiver_OversizedBody413(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	rawKey := fmt.Sprintf("aegis_stow_oversize_%d", time.Now().UnixNano())
	h := sha256.Sum256([]byte(rawKey))
	keyHash := hex.EncodeToString(h[:])
	_, err := model.CreateAPIKey(context.Background(), db, "stow-oversize", keyHash, rawKey[:8], "test", nil)
	require.NoError(t, err)

	dicomData := buildDicomBytes(t, "1.2.3.4.5.413", "MR")
	req := buildStowRequest(t, "/api/stow", dicomData)
	req.Header.Set("Authorization", "Bearer "+rawKey)

	rr := httptest.NewRecorder()
	// Simulate the request body exceeding the cap without allocating 4 GiB:
	// wrap the body in a MaxBytesReader with a tiny limit. The handler's own
	// MaxBytesReader propagates the underlying *http.MaxBytesError from the
	// wrapped reader, which must surface as 413.
	req.Body = http.MaxBytesReader(rr, req.Body, 64)
	srv.StowReceiver(rr, req)

	assert.Equal(t, http.StatusRequestEntityTooLarge, rr.Code, rr.Body.String())
	assert.Contains(t, rr.Body.String(), "too large")
}
