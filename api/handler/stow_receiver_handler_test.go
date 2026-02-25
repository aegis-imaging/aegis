package handler_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"
	"time"

	dicomlib "github.com/suyashkumar/dicom"
	"github.com/suyashkumar/dicom/pkg/tag"
	"github.com/suyashkumar/dicom/pkg/uid"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildDicomBytes returns a minimal parseable DICOM binary with StudyInstanceUID and Modality.
func buildDicomBytes(t *testing.T, studyUID, modality string) []byte {
	t.Helper()
	elements := make([]*dicomlib.Element, 0, 6)

	sopClass, err := dicomlib.NewElement(tag.MediaStorageSOPClassUID, []string{"1.2.840.10008.5.1.4.1.1.4"})
	require.NoError(t, err)

	sopInst, err := dicomlib.NewElement(tag.MediaStorageSOPInstanceUID, []string{studyUID + ".1.1"})
	require.NoError(t, err)

	ts, err := dicomlib.NewElement(tag.TransferSyntaxUID, []string{uid.ExplicitVRLittleEndian})
	require.NoError(t, err)

	modalityEl, err := dicomlib.NewElement(tag.Modality, []string{modality})
	require.NoError(t, err)

	studyUIDEl, err := dicomlib.NewElement(tag.StudyInstanceUID, []string{studyUID})
	require.NoError(t, err)

	sopInstBody, err := dicomlib.NewElement(tag.SOPInstanceUID, []string{studyUID + ".1.1"})
	require.NoError(t, err)

	elements = append(elements, sopClass, sopInst, ts, modalityEl, studyUIDEl, sopInstBody)

	ds := dicomlib.Dataset{Elements: elements}
	var buf bytes.Buffer
	require.NoError(t, dicomlib.Write(&buf, ds))
	return buf.Bytes()
}

// buildStowRequest builds a multipart/related STOW-RS HTTP request with one DICOM file.
func buildStowRequest(t *testing.T, url string, dicomData []byte) *http.Request {
	t.Helper()
	boundary := "AEGIS_TEST_BOUNDARY"

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	mw.SetBoundary(boundary)

	partHeader := textproto.MIMEHeader{}
	partHeader.Set("Content-Type", "application/dicom")
	pw, err := mw.CreatePart(partHeader)
	require.NoError(t, err)
	_, err = pw.Write(dicomData)
	require.NoError(t, err)
	require.NoError(t, mw.Close())

	req := httptest.NewRequest(http.MethodPost, url, &body)
	req.Header.Set("Content-Type", "multipart/related; type=\"application/dicom\"; boundary="+boundary)
	return req
}

// ─── Auth validation ──────────────────────────────────────────────────────────

func TestStowReceiver_NoAuthHeader(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodPost, "/api/stow", bytes.NewReader(nil))
	req.Header.Set("Content-Type", "multipart/related; type=\"application/dicom\"; boundary=XXX")
	rr := httptest.NewRecorder()
	srv.StowReceiver(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Contains(t, rr.Body.String(), "missing API key")
}

func TestStowReceiver_InvalidAPIKey(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodPost, "/api/stow", bytes.NewReader(nil))
	req.Header.Set("Content-Type", "multipart/related; type=\"application/dicom\"; boundary=XXX")
	req.Header.Set("Authorization", "Bearer aegis_bad_key_that_doesnt_exist")
	rr := httptest.NewRecorder()
	srv.StowReceiver(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Contains(t, rr.Body.String(), "invalid or disabled API key")
}

// ─── Content validation ───────────────────────────────────────────────────────

func TestStowReceiver_InvalidContentType(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	// Insert a valid API key.
	rawKey := fmt.Sprintf("aegis_stow_test_%d", time.Now().UnixNano())
	h := sha256.Sum256([]byte(rawKey))
	keyHash := hex.EncodeToString(h[:])
	_, err := model.CreateAPIKey(context.Background(), db, "stow-test", keyHash, rawKey[:8], "test", nil)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/stow", bytes.NewReader([]byte("body")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+rawKey)
	rr := httptest.NewRecorder()
	srv.StowReceiver(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "multipart/related")
}

func TestStowReceiver_MissingBoundary(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	rawKey := fmt.Sprintf("aegis_stow_nobndry_%d", time.Now().UnixNano())
	h := sha256.Sum256([]byte(rawKey))
	keyHash := hex.EncodeToString(h[:])
	_, err := model.CreateAPIKey(context.Background(), db, "stow-no-boundary", keyHash, rawKey[:8], "test", nil)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/stow", bytes.NewReader(nil))
	req.Header.Set("Content-Type", "multipart/related; type=\"application/dicom\"") // no boundary
	req.Header.Set("Authorization", "Bearer "+rawKey)
	rr := httptest.NewRecorder()
	srv.StowReceiver(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "boundary")
}

func TestStowReceiver_ProjectNotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	rawKey := fmt.Sprintf("aegis_stow_projtest_%d", time.Now().UnixNano())
	h := sha256.Sum256([]byte(rawKey))
	keyHash := hex.EncodeToString(h[:])
	_, err := model.CreateAPIKey(context.Background(), db, "stow-proj-test", keyHash, rawKey[:8], "test", nil)
	require.NoError(t, err)

	dcm := buildDicomBytes(t, "1.2.3.4.5.6.7", "MR")
	req := buildStowRequest(t, "/api/stow?project=nonexistent-project", dcm)
	req.Header.Set("Authorization", "Bearer "+rawKey)
	rr := httptest.NewRecorder()
	srv.StowReceiver(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "project not found")
}

func TestStowReceiver_EmptyBody_NoParts(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	rawKey := fmt.Sprintf("aegis_stow_empty_%d", time.Now().UnixNano())
	h := sha256.Sum256([]byte(rawKey))
	keyHash := hex.EncodeToString(h[:])
	_, err := model.CreateAPIKey(context.Background(), db, "stow-empty", keyHash, rawKey[:8], "test", nil)
	require.NoError(t, err)

	// Build an empty multipart body (no parts).
	boundary := "EMPTY_BOUNDARY"
	body := fmt.Sprintf("--%s--\r\n", boundary)
	req := httptest.NewRequest(http.MethodPost, "/api/stow", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "multipart/related; type=\"application/dicom\"; boundary="+boundary)
	req.Header.Set("Authorization", "Bearer "+rawKey)
	rr := httptest.NewRecorder()
	srv.StowReceiver(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "no DICOM parts")
}

func TestStowReceiver_InvalidDICOM(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	rawKey := fmt.Sprintf("aegis_stow_invalid_%d", time.Now().UnixNano())
	h := sha256.Sum256([]byte(rawKey))
	keyHash := hex.EncodeToString(h[:])
	_, err := model.CreateAPIKey(context.Background(), db, "stow-invalid-dcm", keyHash, rawKey[:8], "test", nil)
	require.NoError(t, err)

	// Send a real multipart but with non-DICOM content.
	boundary := "INVALID_DCM_BOUNDARY"
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	mw.SetBoundary(boundary)
	partHeader := textproto.MIMEHeader{}
	partHeader.Set("Content-Type", "application/dicom")
	pw, err := mw.CreatePart(partHeader)
	require.NoError(t, err)
	_, err = pw.Write([]byte("this is not a DICOM file"))
	require.NoError(t, err)
	require.NoError(t, mw.Close())

	req := httptest.NewRequest(http.MethodPost, "/api/stow", &body)
	req.Header.Set("Content-Type", "multipart/related; type=\"application/dicom\"; boundary="+boundary)
	req.Header.Set("Authorization", "Bearer "+rawKey)
	rr := httptest.NewRecorder()
	srv.StowReceiver(rr, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rr.Code)
	assert.Contains(t, rr.Body.String(), "not a valid DICOM file")
}

func TestStowReceiver_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	rawKey := fmt.Sprintf("aegis_stow_ok_%d", time.Now().UnixNano())
	h := sha256.Sum256([]byte(rawKey))
	keyHash := hex.EncodeToString(h[:])
	_, err := model.CreateAPIKey(context.Background(), db, "stow-success-key", keyHash, rawKey[:8], "test", nil)
	require.NoError(t, err)

	studyUID := fmt.Sprintf("1.2.3.stow.%d", time.Now().UnixNano())
	dcm := buildDicomBytes(t, studyUID, "MR")

	req := buildStowRequest(t, "/api/stow", dcm)
	req.Header.Set("Authorization", "Bearer "+rawKey)
	rr := httptest.NewRecorder()
	srv.StowReceiver(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Header().Get("Content-Type"), "application/dicom+json")

	// Verify study was created in the DB.
	study, err := model.GetStudyByUID(context.Background(), db, studyUID)
	require.NoError(t, err)
	assert.Equal(t, studyUID, study.StudyInstanceUID)
	assert.Equal(t, "received", study.Status)
	assert.Equal(t, "MR", study.Modality)
	assert.Equal(t, 1, study.InstanceCount)
}

func TestStowReceiver_MultipleStudies_SingleRequest(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	rawKey := fmt.Sprintf("aegis_stow_multi_%d", time.Now().UnixNano())
	h := sha256.Sum256([]byte(rawKey))
	keyHash := hex.EncodeToString(h[:])
	_, err := model.CreateAPIKey(context.Background(), db, "stow-multi-study", keyHash, rawKey[:8], "test", nil)
	require.NoError(t, err)

	uid1 := fmt.Sprintf("1.2.3.stow.multi1.%d", time.Now().UnixNano())
	uid2 := fmt.Sprintf("1.2.3.stow.multi2.%d", time.Now().UnixNano()+1)

	// Build multipart body with two different studies.
	boundary := "MULTI_STUDY_BOUNDARY"
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	mw.SetBoundary(boundary)

	for _, sUID := range []string{uid1, uid2} {
		ph := textproto.MIMEHeader{}
		ph.Set("Content-Type", "application/dicom")
		pw, werr := mw.CreatePart(ph)
		require.NoError(t, werr)
		_, werr = pw.Write(buildDicomBytes(t, sUID, "CT"))
		require.NoError(t, werr)
	}
	require.NoError(t, mw.Close())

	req := httptest.NewRequest(http.MethodPost, "/api/stow", &body)
	req.Header.Set("Content-Type", "multipart/related; type=\"application/dicom\"; boundary="+boundary)
	req.Header.Set("Authorization", "Bearer "+rawKey)
	rr := httptest.NewRecorder()
	srv.StowReceiver(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	// Both studies should exist.
	s1, err := model.GetStudyByUID(context.Background(), db, uid1)
	require.NoError(t, err)
	assert.Equal(t, uid1, s1.StudyInstanceUID)

	s2, err := model.GetStudyByUID(context.Background(), db, uid2)
	require.NoError(t, err)
	assert.Equal(t, uid2, s2.StudyInstanceUID)
}

func TestStowReceiver_DisabledAPIKey(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	rawKey := fmt.Sprintf("aegis_stow_disabled_%d", time.Now().UnixNano())
	h := sha256.Sum256([]byte(rawKey))
	keyHash := hex.EncodeToString(h[:])
	k, err := model.CreateAPIKey(context.Background(), db, "stow-disabled-key", keyHash, rawKey[:8], "test", nil)
	require.NoError(t, err)

	// Disable the key.
	require.NoError(t, model.UpdateAPIKeyEnabled(context.Background(), db, k.ID, false))

	dcm := buildDicomBytes(t, "1.2.3.disabled", "MR")
	req := buildStowRequest(t, "/api/stow", dcm)
	req.Header.Set("Authorization", "Bearer "+rawKey)
	rr := httptest.NewRecorder()
	srv.StowReceiver(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Contains(t, rr.Body.String(), "invalid or disabled API key")
}
