package handler_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aegis-imaging/aegis/api/config"
	"github.com/aegis-imaging/aegis/api/handler"
	"github.com/aegis-imaging/aegis/api/middleware"
	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/storage"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// authedUploadServer builds a handler.Server with AUTH_ENABLED=true plus the
// production middleware chain for the upload routes (satelliteMTLS →
// uploadAuth → handler), mirroring the wiring in main.go.
func authedUploadServer(t *testing.T, db *sql.DB) (*handler.Server, *config.Config) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "aegis-upload-auth-test-*")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	cfg := &config.Config{
		Port:            "0",
		StorageMode:     "local",
		LocalStorageDir: tmpDir,
		APIBaseURL:      "http://localhost:8080",
		PipelineAuto:    false,
		AuthEnabled:     true,
		AuthProvider:    "auto",
		DevUserEmail:    "test@aegis.local",
		AllowedOrigins:  []string{"http://localhost:3000"},
	}
	store := storage.NewLocal(tmpDir, cfg.APIBaseURL)
	return handler.NewServer(db, store, cfg), cfg
}

// uploadInitBody is the minimal valid init payload targeting the default project.
func uploadInitBody(t *testing.T) *bytes.Reader {
	t.Helper()
	b, err := json.Marshal(map[string]any{"file_count": 1, "project_slug": "default"})
	require.NoError(t, err)
	return bytes.NewReader(b)
}

// uploaderSessionCookie provisions an uploader-role account (optionally a
// project_members row) and returns a live session cookie for it.
func uploaderSessionCookie(t *testing.T, db *sql.DB, email string, memberOfProjectID string) *http.Cookie {
	t.Helper()
	u := testutil.CreateTestAdminUser(t, db, email, "uploader")
	if memberOfProjectID != "" {
		require.NoError(t, model.CreateProjectMember(context.Background(), db, &model.ProjectMember{
			ProjectID:   memberOfProjectID,
			AdminUserID: u.ID,
			Role:        "uploader",
		}))
	}
	sess, err := model.CreateUploaderSession(context.Background(), db, u.ID, "127.0.0.1", "go-test", time.Hour)
	require.NoError(t, err)
	return &http.Cookie{Name: middleware.UploaderSessionCookieName, Value: sess.ID}
}

func TestUploadAuth_AnonymousInitRejected(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv, cfg := authedUploadServer(t, db)
	wrapped := middleware.WithSatelliteMTLS(db)(middleware.RequireUploadAuth(db, cfg)(srv.UploadInit))

	req := httptest.NewRequest(http.MethodPost, "/api/upload/init", uploadInitBody(t))
	rr := httptest.NewRecorder()
	wrapped(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code, rr.Body.String())
	assert.Contains(t, rr.Body.String(), "authentication required")
}

func TestUploadAuth_UploaderWithMembershipAllowed(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv, cfg := authedUploadServer(t, db)
	proj := testutil.SeedProject(t, db)
	cookie := uploaderSessionCookie(t, db, "member-uploader@site.org", proj.ID)
	wrapped := middleware.RequireUploadAuth(db, cfg)(srv.UploadInit)

	req := httptest.NewRequest(http.MethodPost, "/api/upload/init", uploadInitBody(t))
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	wrapped(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	var result map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.NotEmpty(t, result["session_id"])
}

func TestUploadAuth_UploaderWithoutMembershipForbidden(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv, cfg := authedUploadServer(t, db)
	cookie := uploaderSessionCookie(t, db, "nonmember-uploader@site.org", "")
	wrapped := middleware.RequireUploadAuth(db, cfg)(srv.UploadInit)

	req := httptest.NewRequest(http.MethodPost, "/api/upload/init", uploadInitBody(t))
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	wrapped(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code, rr.Body.String())
	assert.Contains(t, rr.Body.String(), "you don't have upload access to this project")
}

func TestUploadAuth_SatelliteBypassesSession(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv, cfg := authedUploadServer(t, db)
	inst := testutil.CreateTestInstitution(t, db, "upload-auth-satellite")
	proj := testutil.SeedProject(t, db)
	require.NoError(t, model.AddInstitutionToProject(context.Background(), db, &model.InstitutionProject{
		InstitutionID: inst.ID, ProjectID: proj.ID, Role: "sender",
	}))
	wrapped := middleware.RequireUploadAuth(db, cfg)(srv.UploadInit)

	req := httptest.NewRequest(http.MethodPost, "/api/upload/init", uploadInitBody(t))
	req = withSatelliteCtx(req, inst) // helper from upload_satellite_attribution_test.go
	rr := httptest.NewRecorder()
	wrapped(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
}

func TestUploadAuth_APIKeyAllowed(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv, cfg := authedUploadServer(t, db)
	owner := testutil.CreateTestAdminUser(t, db, "key-owner@aegis.local", "admin")

	rawKey := "aegis_test_upload_enforce_key"
	hash := sha256.Sum256([]byte(rawKey))
	_, err := model.CreateAPIKey(context.Background(), db, "upload-enforce-test",
		fmt.Sprintf("%x", hash), rawKey[:8], owner.Email, nil)
	require.NoError(t, err)

	wrapped := middleware.RequireUploadAuth(db, cfg)(srv.UploadInit)
	req := httptest.NewRequest(http.MethodPost, "/api/upload/init", uploadInitBody(t))
	req.Header.Set("Authorization", "Bearer "+rawKey)
	rr := httptest.NewRecorder()
	wrapped(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
}

func TestUploadAuth_InvalidAPIKeyRejected(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv, cfg := authedUploadServer(t, db)
	wrapped := middleware.RequireUploadAuth(db, cfg)(srv.UploadInit)

	req := httptest.NewRequest(http.MethodPost, "/api/upload/init", uploadInitBody(t))
	req.Header.Set("Authorization", "Bearer aegis_not_a_real_key")
	rr := httptest.NewRecorder()
	wrapped(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code, rr.Body.String())
}

func TestUploadAuth_FileAndCompleteRequireidentity(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv, cfg := authedUploadServer(t, db)
	uploadAuth := middleware.RequireUploadAuth(db, cfg)

	fileReq := httptest.NewRequest(http.MethodPut, "/api/upload/file/some-session/0", strings.NewReader("data"))
	fileReq.SetPathValue("sessionID", "some-session")
	fileReq.SetPathValue("index", "0")
	rr := httptest.NewRecorder()
	uploadAuth(srv.UploadFile)(rr, fileReq)
	assert.Equal(t, http.StatusUnauthorized, rr.Code, rr.Body.String())

	body, _ := json.Marshal(map[string]any{"session_id": "some-session"})
	completeReq := httptest.NewRequest(http.MethodPost, "/api/upload/complete", bytes.NewReader(body))
	rr = httptest.NewRecorder()
	uploadAuth(srv.UploadComplete)(rr, completeReq)
	assert.Equal(t, http.StatusUnauthorized, rr.Code, rr.Body.String())
}

func TestUploadAuth_SessionScopedFileAndCompleteAllowed(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv, cfg := authedUploadServer(t, db)
	proj := testutil.SeedProject(t, db)
	cookie := uploaderSessionCookie(t, db, "flow-uploader@site.org", proj.ID)
	uploadAuth := middleware.RequireUploadAuth(db, cfg)

	// Init with the session cookie.
	initReq := httptest.NewRequest(http.MethodPost, "/api/upload/init", uploadInitBody(t))
	initReq.AddCookie(cookie)
	rr := httptest.NewRecorder()
	uploadAuth(srv.UploadInit)(rr, initReq)
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	var initResult map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&initResult))
	sessionID := initResult["session_id"].(string)

	// Upload one file with the cookie.
	fileReq := httptest.NewRequest(http.MethodPut, "/api/upload/file/"+sessionID+"/0",
		strings.NewReader("DICM\x00fake-dicom-content"))
	fileReq.SetPathValue("sessionID", sessionID)
	fileReq.SetPathValue("index", "0")
	fileReq.AddCookie(cookie)
	rr = httptest.NewRecorder()
	uploadAuth(srv.UploadFile)(rr, fileReq)
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

	// Complete with the cookie.
	body, _ := json.Marshal(map[string]any{"session_id": sessionID})
	completeReq := httptest.NewRequest(http.MethodPost, "/api/upload/complete", bytes.NewReader(body))
	completeReq.AddCookie(cookie)
	rr = httptest.NewRecorder()
	uploadAuth(srv.UploadComplete)(rr, completeReq)
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
}

func TestUploadAuth_DisabledAuthStaysOpen(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv, _ := dicomTestServer(t, db) // AuthEnabled=false server
	openCfg := &config.Config{AuthEnabled: false}
	wrapped := middleware.RequireUploadAuth(db, openCfg)(srv.UploadInit)

	req := httptest.NewRequest(http.MethodPost, "/api/upload/init", uploadInitBody(t))
	rr := httptest.NewRecorder()
	wrapped(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
}
