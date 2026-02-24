package handler_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── QIDO-RS: Studies ────────────────────────────────────────────────────────

func TestDicomwebStudies_Empty(t *testing.T) {
	db := testutil.TestDB(t)
	srv, _ := dicomTestServer(t, db)

	req := httptest.NewRequest(http.MethodGet, "/dicomweb/studies", nil)
	rr := httptest.NewRecorder()

	srv.DicomwebStudies(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Header().Get("Content-Type"), "application/dicom+json")

	var result []map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.Empty(t, result)
}

func TestDicomwebStudies_ReturnsAll(t *testing.T) {
	db := testutil.TestDB(t)
	srv, _ := dicomTestServer(t, db)
	proj := testutil.SeedProject(t, db)

	testutil.CreateTestStudy(t, db, proj.ID)
	testutil.CreateTestStudy(t, db, proj.ID)

	req := httptest.NewRequest(http.MethodGet, "/dicomweb/studies", nil)
	rr := httptest.NewRecorder()

	srv.DicomwebStudies(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var result []map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.Len(t, result, 2)

	// Each result should have a StudyInstanceUID entry.
	assert.Contains(t, result[0], "0020000D")
}

func TestDicomwebStudies_FilterByUID(t *testing.T) {
	db := testutil.TestDB(t)
	srv, _ := dicomTestServer(t, db)
	proj := testutil.SeedProject(t, db)

	s1 := testutil.CreateTestStudy(t, db, proj.ID)
	testutil.CreateTestStudy(t, db, proj.ID) // second study, should not appear

	req := httptest.NewRequest(http.MethodGet, "/dicomweb/studies?StudyInstanceUIDs="+s1.StudyInstanceUID, nil)
	rr := httptest.NewRecorder()

	srv.DicomwebStudies(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var result []map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	require.Len(t, result, 1)

	// Verify the UID matches.
	uidTag := result[0]["0020000D"].(map[string]any)
	vals := uidTag["Value"].([]any)
	assert.Equal(t, s1.StudyInstanceUID, vals[0])
}

func TestDicomwebStudies_FilterByUID_NotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv, _ := dicomTestServer(t, db)

	req := httptest.NewRequest(http.MethodGet, "/dicomweb/studies?StudyInstanceUIDs=1.2.3.no-such-uid", nil)
	rr := httptest.NewRecorder()

	srv.DicomwebStudies(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var result []map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.Empty(t, result)
}

func TestDicomwebStudies_PatientDataAnonymized(t *testing.T) {
	db := testutil.TestDB(t)
	srv, _ := dicomTestServer(t, db)
	proj := testutil.SeedProject(t, db)
	testutil.CreateTestStudy(t, db, proj.ID)

	req := httptest.NewRequest(http.MethodGet, "/dicomweb/studies", nil)
	rr := httptest.NewRecorder()
	srv.DicomwebStudies(rr, req)

	var result []map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	require.Len(t, result, 1)

	// PatientName (00100010) and PatientID (00100020) should be present but empty.
	patientName := result[0]["00100010"].(map[string]any)
	assert.Equal(t, "PN", patientName["vr"])
	assert.Nil(t, patientName["Value"]) // anonymized — no value

	patientID := result[0]["00100020"].(map[string]any)
	assert.Equal(t, "LO", patientID["vr"])
	assert.Nil(t, patientID["Value"]) // anonymized — no value
}

// ─── QIDO-RS: Series ─────────────────────────────────────────────────────────

func TestDicomwebSeries_NotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv, _ := dicomTestServer(t, db)

	req := httptest.NewRequest(http.MethodGet, "/dicomweb/studies/1.2.3.missing/series", nil)
	req.SetPathValue("studyUID", "1.2.3.missing")
	rr := httptest.NewRecorder()

	srv.DicomwebSeries(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Contains(t, rr.Body.String(), "study not found")
}

func TestDicomwebSeries_ReturnsSingleFakeSeries(t *testing.T) {
	db := testutil.TestDB(t)
	srv, _ := dicomTestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	req := httptest.NewRequest(http.MethodGet, "/dicomweb/studies/"+study.StudyInstanceUID+"/series", nil)
	req.SetPathValue("studyUID", study.StudyInstanceUID)
	rr := httptest.NewRecorder()

	srv.DicomwebSeries(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Header().Get("Content-Type"), "application/dicom+json")

	var result []map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	require.Len(t, result, 1) // always exactly one series per study

	// SeriesInstanceUID should be {studyUID}.1
	seriesTag := result[0]["0020000E"].(map[string]any)
	vals := seriesTag["Value"].([]any)
	assert.Equal(t, study.StudyInstanceUID+".1", vals[0])
}

// ─── QIDO-RS: Instances ──────────────────────────────────────────────────────

func TestDicomwebInstances_NotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv, _ := dicomTestServer(t, db)

	req := httptest.NewRequest(http.MethodGet, "/dicomweb/studies/1.2.3.missing/series/1.2.3.missing.1/instances", nil)
	req.SetPathValue("studyUID", "1.2.3.missing")
	rr := httptest.NewRecorder()

	srv.DicomwebInstances(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestDicomwebInstances_ReturnsAll(t *testing.T) {
	db := testutil.TestDB(t)
	srv, _ := dicomTestServer(t, db)
	proj := testutil.SeedProject(t, db)

	// Create a study with a known instance count.
	study := &model.Study{
		ProjectID:        proj.ID,
		StudyInstanceUID: fmt.Sprintf("1.2.3.dcm-instances-test.%d", uniqueInt()),
		Modality:         "MR",
		InstanceCount:    3,
		Status:           "received",
		DicomStore:       "raw",
		Source:           "external",
	}
	require.NoError(t, model.CreateStudy(context.Background(), db, study))

	req := httptest.NewRequest(http.MethodGet, "/dicomweb/studies/"+study.StudyInstanceUID+"/series/x/instances", nil)
	req.SetPathValue("studyUID", study.StudyInstanceUID)
	rr := httptest.NewRecorder()

	srv.DicomwebInstances(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var result []map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.Len(t, result, 3) // InstanceCount == 3

	// SOPInstanceUIDs should be {studyUID}.1.0, .1.1, .1.2
	sopTag := result[0]["00080018"].(map[string]any)
	vals := sopTag["Value"].([]any)
	assert.Equal(t, study.StudyInstanceUID+".1.0", vals[0])
}

// ─── WADO-RS: Retrieve Instance ──────────────────────────────────────────────

func TestDicomwebRetrieve_InvalidSopUID(t *testing.T) {
	db := testutil.TestDB(t)
	srv, _ := dicomTestServer(t, db)

	req := httptest.NewRequest(http.MethodGet, "/dicomweb/studies/1.2.3/series/s/instances/not-a-number", nil)
	req.SetPathValue("studyUID", "1.2.3")
	req.SetPathValue("sopUID", "not-a-number")
	rr := httptest.NewRecorder()

	srv.DicomwebRetrieveInstance(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "invalid SOP UID")
}

func TestDicomwebRetrieve_StudyNotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv, _ := dicomTestServer(t, db)

	req := httptest.NewRequest(http.MethodGet, "/dicomweb/studies/1.2.3/series/s/instances/1.2.3.1.0", nil)
	req.SetPathValue("studyUID", "1.2.3.no-such-study")
	req.SetPathValue("sopUID", "1.2.3.1.0")
	rr := httptest.NewRecorder()

	srv.DicomwebRetrieveInstance(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Contains(t, rr.Body.String(), "study not found")
}

func TestDicomwebRetrieve_FileNotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv, _ := dicomTestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	// No file written to storage → should 404.
	sopUID := study.StudyInstanceUID + ".1.0"
	req := httptest.NewRequest(http.MethodGet, "/dicomweb/studies/"+study.StudyInstanceUID+"/series/s/instances/"+sopUID, nil)
	req.SetPathValue("studyUID", study.StudyInstanceUID)
	req.SetPathValue("sopUID", sopUID)
	rr := httptest.NewRecorder()

	srv.DicomwebRetrieveInstance(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Contains(t, rr.Body.String(), "file not found")
}

func TestDicomwebRetrieve_Success(t *testing.T) {
	db := testutil.TestDB(t)
	srv, store := dicomTestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	// Write a fake DICOM file at the expected storage key.
	fileKey := "dicom/raw/" + study.StudyInstanceUID + "/0.dcm"
	require.NoError(t, store.Store(context.Background(), fileKey, strings.NewReader("DICM\x00fake")))

	sopUID := study.StudyInstanceUID + ".1.0"

	t.Run("multipart when Accept requests it", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/dicomweb/studies/"+study.StudyInstanceUID+"/series/s/instances/"+sopUID, nil)
		req.Header.Set("Accept", `multipart/related; type="application/dicom"`)
		req.SetPathValue("studyUID", study.StudyInstanceUID)
		req.SetPathValue("sopUID", sopUID)
		rr := httptest.NewRecorder()
		srv.DicomwebRetrieveInstance(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Header().Get("Content-Type"), "multipart/related")
		assert.Contains(t, rr.Body.String(), "DICM")
	})

	t.Run("raw bytes when no multipart Accept", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/dicomweb/studies/"+study.StudyInstanceUID+"/series/s/instances/"+sopUID, nil)
		req.SetPathValue("studyUID", study.StudyInstanceUID)
		req.SetPathValue("sopUID", sopUID)
		rr := httptest.NewRecorder()
		srv.DicomwebRetrieveInstance(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, "application/dicom", rr.Header().Get("Content-Type"))
		assert.Contains(t, rr.Body.String(), "DICM")
	})
}

func TestDicomwebRawRetrieve_AlwaysReadsRawStore(t *testing.T) {
	db := testutil.TestDB(t)
	srv, store := dicomTestServer(t, db)
	proj := testutil.SeedProject(t, db)

	// Study with DicomStore = "clean" (post-defacing).
	study := &model.Study{
		ProjectID:        proj.ID,
		StudyInstanceUID: fmt.Sprintf("1.2.3.raw-override.%d", uniqueInt()),
		Modality:         "MR",
		InstanceCount:    1,
		Status:           "defaced",
		DicomStore:       "clean", // study is in clean store
		Source:           "external",
	}
	require.NoError(t, model.CreateStudy(context.Background(), db, study))

	// Write the file only in the raw store (not clean).
	rawKey := "dicom/raw/" + study.StudyInstanceUID + "/0.dcm"
	require.NoError(t, store.Store(context.Background(), rawKey, strings.NewReader("DICM\x00pre-defacing")))

	sopUID := study.StudyInstanceUID + ".1.0"
	req := httptest.NewRequest(http.MethodGet, "/dicomweb-raw/studies/"+study.StudyInstanceUID+"/series/s/instances/"+sopUID, nil)
	req.SetPathValue("studyUID", study.StudyInstanceUID)
	req.SetPathValue("sopUID", sopUID)
	rr := httptest.NewRecorder()

	// Raw retrieve always reads from "raw", even though study.DicomStore = "clean".
	srv.DicomwebRawRetrieveInstance(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "pre-defacing")
}

// ─── ServeStorageFile ────────────────────────────────────────────────────────

func TestServeStorageFile_Success(t *testing.T) {
	db := testutil.TestDB(t)
	srv, store := dicomTestServer(t, db)

	// Write a test file.
	require.NoError(t, store.Store(context.Background(), "test-file.dcm", strings.NewReader("DICM\x00data")))

	req := httptest.NewRequest(http.MethodGet, "/api/storage/test-file.dcm", nil)
	req.SetPathValue("key", "test-file.dcm")
	rr := httptest.NewRecorder()

	srv.ServeStorageFile(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/dicom", rr.Header().Get("Content-Type"))
	assert.Contains(t, rr.Body.String(), "DICM")
}

func TestServeStorageFile_NotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv, _ := dicomTestServer(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/storage/no-such-file.dcm", nil)
	req.SetPathValue("key", "no-such-file.dcm")
	rr := httptest.NewRecorder()

	srv.ServeStorageFile(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Contains(t, rr.Body.String(), "file not found")
}

func TestServeStorageFile_TraversalBlocked(t *testing.T) {
	db := testutil.TestDB(t)
	srv, _ := dicomTestServer(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/storage/../../etc/passwd", nil)
	req.SetPathValue("key", "../../etc/passwd")
	rr := httptest.NewRecorder()

	srv.ServeStorageFile(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "invalid path")
}

// ─── WADO-RS: Instance Metadata ──────────────────────────────────────────────

func TestDicomwebInstanceMetadata_InvalidSopUID(t *testing.T) {
	db := testutil.TestDB(t)
	srv, _ := dicomTestServer(t, db)

	req := httptest.NewRequest(http.MethodGet, "/dicomweb/studies/1.2.3/series/s/instances/not-a-number/metadata", nil)
	req.SetPathValue("studyUID", "1.2.3")
	req.SetPathValue("sopUID", "not-a-number")
	rr := httptest.NewRecorder()

	srv.DicomwebInstanceMetadata(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "invalid SOP UID")
}

func TestDicomwebInstanceMetadata_StudyNotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv, _ := dicomTestServer(t, db)

	req := httptest.NewRequest(http.MethodGet, "/dicomweb/studies/1.2.3.no-such/series/s/instances/1.2.3.1.0/metadata", nil)
	req.SetPathValue("studyUID", "1.2.3.no-such")
	req.SetPathValue("sopUID", "1.2.3.1.0")
	rr := httptest.NewRecorder()

	srv.DicomwebInstanceMetadata(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Contains(t, rr.Body.String(), "study not found")
}

func TestDicomwebInstanceMetadata_FileNotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv, _ := dicomTestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	sopUID := study.StudyInstanceUID + ".1.0"
	req := httptest.NewRequest(http.MethodGet, "/dicomweb/studies/"+study.StudyInstanceUID+"/series/s/instances/"+sopUID+"/metadata", nil)
	req.SetPathValue("studyUID", study.StudyInstanceUID)
	req.SetPathValue("sopUID", sopUID)
	rr := httptest.NewRecorder()

	srv.DicomwebInstanceMetadata(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Contains(t, rr.Body.String(), "file not found")
}

func TestDicomwebInstanceMetadata_InvalidDicom(t *testing.T) {
	db := testutil.TestDB(t)
	srv, store := dicomTestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	// Write non-parseable content.
	writeFakeDcm(t, store, "dicom/raw/"+study.StudyInstanceUID+"/0.dcm")

	sopUID := study.StudyInstanceUID + ".1.0"
	req := httptest.NewRequest(http.MethodGet, "/dicomweb/studies/"+study.StudyInstanceUID+"/series/s/instances/"+sopUID+"/metadata", nil)
	req.SetPathValue("studyUID", study.StudyInstanceUID)
	req.SetPathValue("sopUID", sopUID)
	rr := httptest.NewRecorder()

	srv.DicomwebInstanceMetadata(rr, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rr.Code)
	assert.Contains(t, rr.Body.String(), "failed to parse DICOM file")
}

func TestDicomwebInstanceMetadata_Success(t *testing.T) {
	db := testutil.TestDB(t)
	srv, store := dicomTestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	// Write a real parseable DICOM file with Modality=MR.
	writeRealDcm(t, store, "dicom/raw/"+study.StudyInstanceUID+"/0.dcm", "MR")

	sopUID := study.StudyInstanceUID + ".1.0"
	req := httptest.NewRequest(http.MethodGet, "/dicomweb/studies/"+study.StudyInstanceUID+"/series/s/instances/"+sopUID+"/metadata", nil)
	req.SetPathValue("studyUID", study.StudyInstanceUID)
	req.SetPathValue("sopUID", sopUID)
	rr := httptest.NewRecorder()

	srv.DicomwebInstanceMetadata(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Header().Get("Content-Type"), "application/dicom+json")

	// Response must be a JSON array with exactly one object.
	var result []map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	require.Len(t, result, 1)

	// Modality tag (0008,0060) should be present with value "MR".
	modalityTag, ok := result[0]["00080060"]
	require.True(t, ok, "Modality tag 00080060 should be present")
	modalityObj := modalityTag.(map[string]any)
	assert.Equal(t, "CS", modalityObj["vr"])
	vals := modalityObj["Value"].([]any)
	assert.Equal(t, "MR", vals[0])
}

func TestDicomwebRawInstanceMetadata_AlwaysReadsRawStore(t *testing.T) {
	db := testutil.TestDB(t)
	srv, store := dicomTestServer(t, db)
	proj := testutil.SeedProject(t, db)

	// Study with DicomStore = "clean" (post-defacing).
	study := &model.Study{
		ProjectID:        proj.ID,
		StudyInstanceUID: fmt.Sprintf("1.2.3.meta-raw-override.%d", uniqueInt()),
		Modality:         "CT",
		InstanceCount:    1,
		Status:           "defaced",
		DicomStore:       "clean",
		Source:           "external",
	}
	require.NoError(t, model.CreateStudy(context.Background(), db, study))

	// Write the real DICOM file only in the raw store.
	writeRealDcm(t, store, "dicom/raw/"+study.StudyInstanceUID+"/0.dcm", "CT")

	sopUID := study.StudyInstanceUID + ".1.0"
	req := httptest.NewRequest(http.MethodGet, "/dicomweb-raw/studies/"+study.StudyInstanceUID+"/series/s/instances/"+sopUID+"/metadata", nil)
	req.SetPathValue("studyUID", study.StudyInstanceUID)
	req.SetPathValue("sopUID", sopUID)
	rr := httptest.NewRecorder()

	// Raw metadata always reads from "raw", even though study.DicomStore = "clean".
	srv.DicomwebRawInstanceMetadata(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var result []map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	require.Len(t, result, 1)

	// Modality tag should reflect the raw file's CT modality.
	modalityTag, ok := result[0]["00080060"]
	require.True(t, ok, "Modality tag should be present")
	modalityObj := modalityTag.(map[string]any)
	vals := modalityObj["Value"].([]any)
	assert.Equal(t, "CT", vals[0])
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

var uniqueCounter int

// uniqueInt returns a unique integer for generating distinct study UIDs in tests.
func uniqueInt() int {
	uniqueCounter++
	return uniqueCounter
}
