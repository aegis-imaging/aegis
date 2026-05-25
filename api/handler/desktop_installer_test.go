package handler_test

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aegis-imaging/aegis/api/handler"
	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// helper: register an installer via the JSON external_url path (no multipart, no storage hit).
func createExternalInstaller(t *testing.T, srv *handler.Server, product, platform, version, externalURL string) string {
	t.Helper()
	body, _ := json.Marshal(map[string]any{
		"product":      product,
		"platform":     platform,
		"version":      version,
		"external_url": externalURL,
		"filename":     "test-installer.dmg",
		"size_bytes":   12345,
		"sha256":       "deadbeef",
		"changelog":    "Test release",
		"mark_current": true,
	})
	req := httptest.NewRequest("POST", "/api/desktop-installers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.CreateDesktopInstaller(rr, req)
	require.Equal(t, http.StatusCreated, rr.Code, "create installer: %s", rr.Body.String())
	var out model.DesktopInstaller
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&out))
	return out.ID
}

func TestCreateDesktopInstaller_External(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body, _ := json.Marshal(map[string]any{
		"product":      "uploader",
		"platform":     "macos-arm64",
		"version":      "1.0.0",
		"external_url": "https://example.com/aegis-uploader-1.0.0.dmg",
		"filename":     "aegis-uploader-1.0.0.dmg",
		"size_bytes":   10485760,
		"sha256":       "abc123",
		"changelog":    "First release",
		"mark_current": true,
	})
	req := httptest.NewRequest("POST", "/api/desktop-installers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.CreateDesktopInstaller(rr, req)

	require.Equal(t, http.StatusCreated, rr.Code, rr.Body.String())
	var inst model.DesktopInstaller
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&inst))
	assert.Equal(t, "uploader", inst.Product)
	assert.Equal(t, "macos-arm64", inst.Platform)
	assert.Equal(t, "1.0.0", inst.Version)
	assert.Equal(t, "https://example.com/aegis-uploader-1.0.0.dmg", inst.ExternalURL)
	assert.Empty(t, inst.StorageKey)
}

func TestCreateDesktopInstaller_InvalidProduct(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body, _ := json.Marshal(map[string]any{
		"product":      "viewer", // not in allow-list
		"platform":     "macos-arm64",
		"version":      "1.0.0",
		"external_url": "https://example.com/x.dmg",
		"filename":     "x.dmg",
	})
	req := httptest.NewRequest("POST", "/api/desktop-installers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.CreateDesktopInstaller(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestCreateDesktopInstaller_InvalidSemver(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body, _ := json.Marshal(map[string]any{
		"product":      "uploader",
		"platform":     "macos-arm64",
		"version":      "not-a-version",
		"external_url": "https://example.com/x.dmg",
		"filename":     "x.dmg",
	})
	req := httptest.NewRequest("POST", "/api/desktop-installers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.CreateDesktopInstaller(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestCreateDesktopInstaller_DuplicateConflict(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	mkBody := func() []byte {
		b, _ := json.Marshal(map[string]any{
			"product":      "uploader",
			"platform":     "macos-arm64",
			"version":      "1.0.0",
			"external_url": "https://example.com/x.dmg",
			"filename":     "x.dmg",
		})
		return b
	}

	req1 := httptest.NewRequest("POST", "/api/desktop-installers", bytes.NewReader(mkBody()))
	req1.Header.Set("Content-Type", "application/json")
	rr1 := httptest.NewRecorder()
	srv.CreateDesktopInstaller(rr1, req1)
	require.Equal(t, http.StatusCreated, rr1.Code)

	req2 := httptest.NewRequest("POST", "/api/desktop-installers", bytes.NewReader(mkBody()))
	req2.Header.Set("Content-Type", "application/json")
	rr2 := httptest.NewRecorder()
	srv.CreateDesktopInstaller(rr2, req2)
	assert.Equal(t, http.StatusConflict, rr2.Code)
}

func TestCreateDesktopInstaller_MultipartUpload(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	// Build a multipart upload with a tiny "binary" payload.
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	require.NoError(t, mw.WriteField("product", "uploader"))
	require.NoError(t, mw.WriteField("platform", "macos-arm64"))
	require.NoError(t, mw.WriteField("version", "1.0.0"))
	require.NoError(t, mw.WriteField("changelog", "First release"))
	require.NoError(t, mw.WriteField("mark_current", "true"))
	fw, err := mw.CreateFormFile("file", "aegis-uploader-1.0.0.dmg")
	require.NoError(t, err)
	_, err = io.WriteString(fw, "pretend this is a 10MB installer binary")
	require.NoError(t, err)
	require.NoError(t, mw.Close())

	req := httptest.NewRequest("POST", "/api/desktop-installers", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rr := httptest.NewRecorder()
	srv.CreateDesktopInstaller(rr, req)

	require.Equal(t, http.StatusCreated, rr.Code, rr.Body.String())
	var inst model.DesktopInstaller
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&inst))
	assert.Equal(t, "installers/uploader/1.0.0/aegis-uploader-1.0.0.dmg", inst.StorageKey)
	assert.NotEmpty(t, inst.SHA256, "sha256 should be computed during upload")
	assert.True(t, inst.IsCurrent, "mark_current=true should set is_current")
	assert.Equal(t, int64(len("pretend this is a 10MB installer binary")), inst.SizeBytes)
}

func TestListDesktopInstallers(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	createExternalInstaller(t, srv, "uploader", "macos-arm64", "1.0.0", "https://example.com/a.dmg")
	createExternalInstaller(t, srv, "uploader", "windows-x64", "1.0.0", "https://example.com/b.exe")

	req := httptest.NewRequest("GET", "/api/desktop-installers", nil)
	rr := httptest.NewRecorder()
	srv.ListDesktopInstallers(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var resp struct {
		Installers []model.DesktopInstaller `json:"installers"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Len(t, resp.Installers, 2)
}

func TestMarkCurrent_OnlyOneCurrentPerPlatform(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	id1 := createExternalInstaller(t, srv, "uploader", "macos-arm64", "1.0.0", "https://example.com/a.dmg")
	// Second install starts not-current (mark_current is true on create but we
	// then mark a NEWER version current, expecting the old one to flip).
	id2 := createExternalInstaller(t, srv, "uploader", "macos-arm64", "1.1.0", "https://example.com/b.dmg")

	// Mark v1.0.0 (older) current. This should clear v1.1.0's flag.
	body, _ := json.Marshal(map[string]any{"mark_current": true})
	req := httptest.NewRequest("PATCH", "/api/desktop-installers/"+id1, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", id1)
	rr := httptest.NewRecorder()
	srv.UpdateDesktopInstaller(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	// Confirm only id1 is current.
	all, err := model.ListDesktopInstallers(req.Context(), db)
	require.NoError(t, err)
	for _, inst := range all {
		if inst.ID == id1 {
			assert.True(t, inst.IsCurrent, "v1.0.0 should now be current")
		}
		if inst.ID == id2 {
			assert.False(t, inst.IsCurrent, "v1.1.0 should have lost current flag")
		}
	}
}

func TestDeleteDesktopInstaller(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	id := createExternalInstaller(t, srv, "uploader", "macos-arm64", "1.0.0", "https://example.com/a.dmg")

	req := httptest.NewRequest("DELETE", "/api/desktop-installers/"+id, nil)
	req.SetPathValue("id", id)
	rr := httptest.NewRecorder()
	srv.DeleteDesktopInstaller(rr, req)
	assert.Equal(t, http.StatusNoContent, rr.Code)

	// Confirm gone.
	all, err := model.ListDesktopInstallers(req.Context(), db)
	require.NoError(t, err)
	assert.Empty(t, all)
}

func TestEmailInvite_NoSMTPConfigured(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db) // TestServer leaves SMTPHost empty

	id := createExternalInstaller(t, srv, "uploader", "macos-arm64", "1.0.0", "https://example.com/a.dmg")

	body, _ := json.Marshal(map[string]any{"recipient_email": "user@example.com"})
	req := httptest.NewRequest("POST", "/api/desktop-installers/"+id+"/email", bytes.NewReader(body))
	req.SetPathValue("id", id)
	rr := httptest.NewRecorder()
	srv.EmailDesktopInstallerInvite(rr, req)

	assert.Equal(t, http.StatusServiceUnavailable, rr.Code,
		"should refuse to send when SMTP_HOST is unset")
}

func TestPairDesktopInstaller_UnknownToken(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body, _ := json.Marshal(map[string]any{"pairing_token": "totally-bogus"})
	req := httptest.NewRequest("POST", "/api/install/pair", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	srv.PairDesktopInstaller(rr, req)

	assert.Equal(t, http.StatusGone, rr.Code,
		"unknown pairing token should return 410 Gone")
}

func TestPairDesktopInstaller_HappyPath(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	// Bypass the email path: create an invite directly via the model layer so
	// we know the pairing_token. The handler path requires SMTP.
	id := createExternalInstaller(t, srv, "uploader", "macos-arm64", "1.0.0", "https://example.com/a.dmg")
	inv, err := model.CreateDesktopInstallerInvite(t.Context(), db, model.CreateDesktopInstallerInviteInput{
		InstallerID:    id,
		RecipientEmail: "user@example.com",
		ExpiresAt:      time.Now().UTC().Add(7 * 24 * time.Hour),
		SentBy:         "admin@example.com",
	})
	require.NoError(t, err)
	pair := inv.PairingToken()
	require.NotEmpty(t, pair)

	body, _ := json.Marshal(map[string]any{"pairing_token": pair})
	req := httptest.NewRequest("POST", "/api/install/pair", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	srv.PairDesktopInstaller(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	var out struct {
		APIKey    string `json:"api_key"`
		ServerURL string `json:"server_url"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&out))
	assert.True(t, len(out.APIKey) > 10, "should return a real API key")

	// Second call with same token must fail.
	req2 := httptest.NewRequest("POST", "/api/install/pair", bytes.NewReader(body))
	rr2 := httptest.NewRecorder()
	srv.PairDesktopInstaller(rr2, req2)
	assert.Equal(t, http.StatusGone, rr2.Code, "pairing token is single-use")
}

// TestCreateInvite_WithInstitutionID exercises the new institution_id link:
// when an admin emails an invite tied to an institution, the row stores the
// institution_id and the per-institution list returns it.
func TestCreateInvite_WithInstitutionID(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	id := createExternalInstaller(t, srv, "uploader", "macos-arm64", "1.0.0", "https://example.com/a.dmg")
	inst := testutil.CreateTestInstitution(t, db, "uchicago")

	inv, err := model.CreateDesktopInstallerInvite(t.Context(), db, model.CreateDesktopInstallerInviteInput{
		InstallerID:    id,
		InstitutionID:  inst.ID,
		RecipientEmail: "site-admin@uchicago.edu",
		ExpiresAt:      time.Now().UTC().Add(7 * 24 * time.Hour),
		SentBy:         "admin@example.com",
	})
	require.NoError(t, err)
	require.NotNil(t, inv.InstitutionID)
	assert.Equal(t, inst.ID, *inv.InstitutionID)

	// Per-institution handler returns this invite.
	req := httptest.NewRequest("GET", "/api/institutions/"+inst.ID+"/installer-invites", nil)
	req.SetPathValue("id", inst.ID)
	rr := httptest.NewRecorder()
	srv.ListInstitutionInstallerInvites(rr, req)
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

	var out struct {
		Invites []model.DesktopInstallerInvite `json:"invites"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&out))
	require.Len(t, out.Invites, 1)
	assert.Equal(t, inv.ID, out.Invites[0].ID)
	require.NotNil(t, out.Invites[0].InstitutionName)
	assert.Equal(t, inst.Name, *out.Invites[0].InstitutionName,
		"per-institution listing must join the institution name for display")
}

// TestCreateInvite_WithoutInstitutionID — back-compat: existing invites that
// don't name an institution still work and don't appear in any
// per-institution listing.
func TestCreateInvite_WithoutInstitutionID(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	id := createExternalInstaller(t, srv, "uploader", "macos-arm64", "1.0.0", "https://example.com/a.dmg")
	inst := testutil.CreateTestInstitution(t, db, "unrelated-inst")

	inv, err := model.CreateDesktopInstallerInvite(t.Context(), db, model.CreateDesktopInstallerInviteInput{
		InstallerID:    id,
		RecipientEmail: "global@example.com",
		ExpiresAt:      time.Now().UTC().Add(7 * 24 * time.Hour),
		SentBy:         "admin@example.com",
	})
	require.NoError(t, err)
	assert.Nil(t, inv.InstitutionID, "back-compat: institution-less invites store NULL")

	// Per-institution listing for a different institution does not surface it.
	req := httptest.NewRequest("GET", "/api/institutions/"+inst.ID+"/installer-invites", nil)
	req.SetPathValue("id", inst.ID)
	rr := httptest.NewRecorder()
	srv.ListInstitutionInstallerInvites(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
	var out struct {
		Invites []model.DesktopInstallerInvite `json:"invites"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&out))
	assert.Empty(t, out.Invites,
		"an institution should only see invites that name it (not all global invites)")
}

// TestListInstitutionInstallerInvites_UnknownInstitution returns 404 instead
// of leaking via an empty list — keeps the surface predictable for the UI.
func TestListInstitutionInstallerInvites_UnknownInstitution(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest("GET", "/api/institutions/00000000-0000-0000-0000-000000000000/installer-invites", nil)
	req.SetPathValue("id", "00000000-0000-0000-0000-000000000000")
	rr := httptest.NewRecorder()
	srv.ListInstitutionInstallerInvites(rr, req)
	assert.Equal(t, http.StatusNotFound, rr.Code)
}
