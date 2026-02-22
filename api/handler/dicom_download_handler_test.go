package handler_test

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aegis-imaging/aegis/api/config"
	"github.com/aegis-imaging/aegis/api/handler"
	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/storage"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// dicomTestServer creates a handler.Server backed by local storage in a temp dir.
// Returns the server and the storage object so tests can write fake DICOM files.
func dicomTestServer(t *testing.T, db *sql.DB) (*handler.Server, *storage.Local) {
	t.Helper()
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Port:            "0",
		StorageMode:     "local",
		LocalStorageDir: tmpDir,
		APIBaseURL:      "http://localhost:8080",
		PipelineAuto:    false,
		AuthEnabled:     false,
		DevUserEmail:    "test@aegis.local",
		AllowedOrigins:  []string{"http://localhost:3000"},
	}
	store := storage.NewLocal(tmpDir, cfg.APIBaseURL)
	return handler.NewServer(db, store, cfg), store
}

// writeFakeDcm writes minimal fake content to a DICOM key in test storage.
func writeFakeDcm(t *testing.T, store *storage.Local, key string) {
	t.Helper()
	require.NoError(t, store.Store(context.Background(), key, strings.NewReader("DICM\x00fake-dicom-content")))
}

// ─── ServeDicomDownload (admin path) ─────────────────────────────────────────

func TestServeDicomDownload_StudyNotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv, _ := dicomTestServer(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/studies/no-such-study/dicom-download", nil)
	req.SetPathValue("studyUID", "1.2.3.999.no-such-study")
	rr := httptest.NewRecorder()

	srv.ServeDicomDownload(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Contains(t, rr.Body.String(), "study not found")
}

func TestServeDicomDownload_NotApproved(t *testing.T) {
	db := testutil.TestDB(t)
	srv, _ := dicomTestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID) // status=received

	req := httptest.NewRequest(http.MethodGet, "/api/studies/"+study.StudyInstanceUID+"/dicom-download", nil)
	req.SetPathValue("studyUID", study.StudyInstanceUID)
	rr := httptest.NewRecorder()

	srv.ServeDicomDownload(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "only approved studies can be downloaded")
}

func TestServeDicomDownload_NoFiles(t *testing.T) {
	db := testutil.TestDB(t)
	srv, _ := dicomTestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)
	require.NoError(t, model.UpdateStudyStatus(context.Background(), db, study.ID, "approved"))

	req := httptest.NewRequest(http.MethodGet, "/api/studies/"+study.StudyInstanceUID+"/dicom-download", nil)
	req.SetPathValue("studyUID", study.StudyInstanceUID)
	rr := httptest.NewRecorder()

	srv.ServeDicomDownload(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Contains(t, rr.Body.String(), "no DICOM files found")
}

func TestServeDicomDownload_Success(t *testing.T) {
	db := testutil.TestDB(t)
	srv, store := dicomTestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)
	require.NoError(t, model.UpdateStudyStatus(context.Background(), db, study.ID, "approved"))

	// Write two fake DICOM files to storage.
	writeFakeDcm(t, store, "dicom/raw/"+study.StudyInstanceUID+"/0.dcm")
	writeFakeDcm(t, store, "dicom/raw/"+study.StudyInstanceUID+"/1.dcm")

	req := httptest.NewRequest(http.MethodGet, "/api/studies/"+study.StudyInstanceUID+"/dicom-download", nil)
	req.SetPathValue("studyUID", study.StudyInstanceUID)
	rr := httptest.NewRecorder()

	srv.ServeDicomDownload(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/zip", rr.Header().Get("Content-Type"))
	assert.Contains(t, rr.Header().Get("Content-Disposition"), "dicom.zip")

	// Verify the zip archive is well-formed and contains the expected files.
	body := rr.Body.Bytes()
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	require.NoError(t, err)
	require.Len(t, zr.File, 2)
}

// ─── ServeDicomDownloadByToken (export share path) ───────────────────────────

// createShareToken generates a raw token and the corresponding hash to store in DB.
func createShareToken(rawToken string) string {
	h := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(h[:])
}

func TestServeDicomDownloadByToken_NotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv, _ := dicomTestServer(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/export/nosuchtoken/download", nil)
	req.SetPathValue("token", "no-such-token-12345")
	rr := httptest.NewRecorder()

	srv.ServeDicomDownloadByToken(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Contains(t, rr.Body.String(), "share not found")
}

func TestServeDicomDownloadByToken_Expired(t *testing.T) {
	db := testutil.TestDB(t)
	srv, _ := dicomTestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	rawToken := "expired-token-abc"
	tokenHash := createShareToken(rawToken)
	past := time.Now().Add(-48 * time.Hour)
	_, err := model.CreateExportShare(context.Background(), db, study.ID, tokenHash, "r@test.com", "", "admin", past)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/export/"+rawToken+"/download", nil)
	req.SetPathValue("token", rawToken)
	rr := httptest.NewRecorder()

	srv.ServeDicomDownloadByToken(rr, req)

	assert.Equal(t, http.StatusGone, rr.Code)
	assert.Contains(t, rr.Body.String(), "share has expired")
}

func TestServeDicomDownloadByToken_Revoked(t *testing.T) {
	db := testutil.TestDB(t)
	srv, _ := dicomTestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	rawToken := "revoked-token-xyz"
	tokenHash := createShareToken(rawToken)
	future := time.Now().Add(24 * time.Hour)
	share, err := model.CreateExportShare(context.Background(), db, study.ID, tokenHash, "r@test.com", "", "admin", future)
	require.NoError(t, err)
	require.NoError(t, model.RevokeExportShare(context.Background(), db, share.ID))

	req := httptest.NewRequest(http.MethodGet, "/api/export/"+rawToken+"/download", nil)
	req.SetPathValue("token", rawToken)
	rr := httptest.NewRecorder()

	srv.ServeDicomDownloadByToken(rr, req)

	assert.Equal(t, http.StatusGone, rr.Code)
	assert.Contains(t, rr.Body.String(), "share has been revoked")
}

func TestServeDicomDownloadByToken_NoFiles(t *testing.T) {
	db := testutil.TestDB(t)
	srv, _ := dicomTestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	rawToken := "nofiles-token-456"
	tokenHash := createShareToken(rawToken)
	future := time.Now().Add(24 * time.Hour)
	_, err := model.CreateExportShare(context.Background(), db, study.ID, tokenHash, "r@test.com", "", "admin", future)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/export/"+rawToken+"/download", nil)
	req.SetPathValue("token", rawToken)
	rr := httptest.NewRecorder()

	srv.ServeDicomDownloadByToken(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Contains(t, rr.Body.String(), "no DICOM files found")
}

func TestServeDicomDownloadByToken_Success(t *testing.T) {
	db := testutil.TestDB(t)
	srv, store := dicomTestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	// Write a fake DICOM file.
	writeFakeDcm(t, store, "dicom/raw/"+study.StudyInstanceUID+"/0.dcm")

	rawToken := "valid-token-success-789"
	tokenHash := createShareToken(rawToken)
	future := time.Now().Add(24 * time.Hour)
	_, err := model.CreateExportShare(context.Background(), db, study.ID, tokenHash, "recipient@test.com", "test note", "admin", future)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/export/"+rawToken+"/download", nil)
	req.SetPathValue("token", rawToken)
	rr := httptest.NewRecorder()

	srv.ServeDicomDownloadByToken(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/zip", rr.Header().Get("Content-Type"))

	// Verify the zip archive is readable and contains the DICOM file.
	body := rr.Body.Bytes()
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	require.NoError(t, err)
	require.Len(t, zr.File, 1)
	assert.Equal(t, "0.dcm", zr.File[0].Name)

	// Verify the file contents are streamed correctly.
	rc, err := zr.File[0].Open()
	require.NoError(t, err)
	defer rc.Close()
	content, err := io.ReadAll(rc)
	require.NoError(t, err)
	assert.Contains(t, string(content), "DICM")
}
