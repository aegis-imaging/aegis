package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	dicomlib "github.com/suyashkumar/dicom"
	"github.com/suyashkumar/dicom/pkg/tag"
	"github.com/suyashkumar/dicom/pkg/uid"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/storage"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeRealDcmWithPatient writes a parseable DICOM file with Modality and optional PatientName.
// Includes required DICOM meta elements so dicomlib.Write succeeds.
func writeRealDcmWithPatient(t *testing.T, store *storage.Local, key, modality, patientName string) {
	t.Helper()
	sopClassEl, err := dicomlib.NewElement(tag.MediaStorageSOPClassUID, []string{"1.2.840.10008.5.1.4.1.1.4"})
	require.NoError(t, err)
	sopInstanceEl, err := dicomlib.NewElement(tag.MediaStorageSOPInstanceUID, []string{"1.2.3.4.5.6.7.8"})
	require.NoError(t, err)
	transferSyntaxEl, err := dicomlib.NewElement(tag.TransferSyntaxUID, []string{uid.ExplicitVRLittleEndian})
	require.NoError(t, err)
	modalityEl, err := dicomlib.NewElement(tag.Modality, []string{modality})
	require.NoError(t, err)

	elements := []*dicomlib.Element{sopClassEl, sopInstanceEl, transferSyntaxEl, modalityEl}

	if patientName != "" {
		pnEl, err := dicomlib.NewElement(tag.PatientName, []string{patientName})
		require.NoError(t, err)
		elements = append(elements, pnEl)
	}

	ds := dicomlib.Dataset{Elements: elements}
	var buf bytes.Buffer
	require.NoError(t, dicomlib.Write(&buf, ds))
	require.NoError(t, store.Store(context.Background(), key, &buf))
}

func TestGetAnonDiff_StudyNotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv, _ := dicomTestServer(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/studies/9.9.notexist/anonymization-diff", nil)
	req.SetPathValue("studyUID", "9.9.notexist")
	rr := httptest.NewRecorder()
	srv.GetAnonDiff(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestGetAnonDiff_NoRawFiles_404(t *testing.T) {
	db := testutil.TestDB(t)
	srv, _ := dicomTestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	req := httptest.NewRequest(http.MethodGet, "/api/studies/"+study.StudyInstanceUID+"/anonymization-diff", nil)
	req.SetPathValue("studyUID", study.StudyInstanceUID)
	rr := httptest.NewRecorder()
	srv.GetAnonDiff(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestGetAnonDiff_NoCleanFiles_204(t *testing.T) {
	db := testutil.TestDB(t)
	srv, store := dicomTestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	// Write raw file only — no clean store.
	writeRealDcm(t, store, "dicom/raw/"+study.StudyInstanceUID+"/0.dcm", "MR")

	req := httptest.NewRequest(http.MethodGet, "/api/studies/"+study.StudyInstanceUID+"/anonymization-diff", nil)
	req.SetPathValue("studyUID", study.StudyInstanceUID)
	rr := httptest.NewRecorder()
	srv.GetAnonDiff(rr, req)

	assert.Equal(t, http.StatusNoContent, rr.Code)
}

func TestGetAnonDiff_Success_ShowsRemovedTags(t *testing.T) {
	db := testutil.TestDB(t)
	srv, store := dicomTestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	// Raw file has PatientName; clean file does not.
	writeRealDcmWithPatient(t, store, "dicom/raw/"+study.StudyInstanceUID+"/0.dcm", "MR", "SMITH^JOHN")
	writeRealDcm(t, store, "dicom/clean/"+study.StudyInstanceUID+"/0.dcm", "MR")

	req := httptest.NewRequest(http.MethodGet, "/api/studies/"+study.StudyInstanceUID+"/anonymization-diff", nil)
	req.SetPathValue("studyUID", study.StudyInstanceUID)
	rr := httptest.NewRecorder()
	srv.GetAnonDiff(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var resp struct {
		StudyID string `json:"study_id"`
		Diff    struct {
			Removed []struct {
				Keyword string `json:"keyword"`
			} `json:"removed"`
			Modified []any `json:"modified"`
			Added    []any `json:"added"`
		} `json:"diff"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, study.ID, resp.StudyID)

	removedKeywords := make([]string, 0, len(resp.Diff.Removed))
	for _, r := range resp.Diff.Removed {
		removedKeywords = append(removedKeywords, r.Keyword)
	}
	assert.Contains(t, removedKeywords, "PatientName")
}

func TestGetAnonDiff_IdenticalFiles_EmptyDiff(t *testing.T) {
	db := testutil.TestDB(t)
	srv, store := dicomTestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	// Write identical DICOM files to both stores.
	writeRealDcm(t, store, "dicom/raw/"+study.StudyInstanceUID+"/0.dcm", "MR")
	writeRealDcm(t, store, "dicom/clean/"+study.StudyInstanceUID+"/0.dcm", "MR")

	req := httptest.NewRequest(http.MethodGet, "/api/studies/"+study.StudyInstanceUID+"/anonymization-diff", nil)
	req.SetPathValue("studyUID", study.StudyInstanceUID)
	rr := httptest.NewRecorder()
	srv.GetAnonDiff(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var resp struct {
		Diff struct {
			Removed  []any `json:"removed"`
			Modified []any `json:"modified"`
			Added    []any `json:"added"`
		} `json:"diff"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Empty(t, resp.Diff.Removed)
	assert.Empty(t, resp.Diff.Modified)
	assert.Empty(t, resp.Diff.Added)
}

func TestGetAnonDiff_SiteScopedOutOfSiteDenied(t *testing.T) {
	db := testutil.TestDB(t)
	srv, _ := dicomTestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	instA := createInstitution(t, db, "sender", "PACS_DIFF_A", true)
	instB := createInstitution(t, db, "sender", "PACS_DIFF_B", true)

	_, err := db.ExecContext(context.Background(), `UPDATE studies SET institution_id = $1 WHERE id = $2`, instA.ID, study.ID)
	require.NoError(t, err)

	researcher := testutil.CreateTestAdminUser(t, db, "anon-diff-site@test.com", "researcher")
	instBID := instB.ID
	require.NoError(t, model.CreateProjectMember(context.Background(), db, &model.ProjectMember{
		ProjectID:     proj.ID,
		AdminUserID:   researcher.ID,
		Role:          "site_coordinator",
		InstitutionID: &instBID,
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/studies/"+study.StudyInstanceUID+"/anonymization-diff", nil)
	req.SetPathValue("studyUID", study.StudyInstanceUID)
	req = withResearcherUser(req, researcher.ID, researcher.Email)
	rr := httptest.NewRecorder()

	srv.GetAnonDiff(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Contains(t, rr.Body.String(), "study not found")
}
