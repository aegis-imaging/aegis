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

// writeRealDcm writes a minimal, parseable DICOM file to test storage using the
// suyashkumar/dicom write API. The dataset contains Modality=<modality>.
func writeRealDcm(t *testing.T, store *storage.Local, key, modality string) {
	t.Helper()

	sopClassEl, err := dicomlib.NewElement(tag.MediaStorageSOPClassUID, []string{"1.2.840.10008.5.1.4.1.1.4"})
	require.NoError(t, err)
	sopInstanceEl, err := dicomlib.NewElement(tag.MediaStorageSOPInstanceUID, []string{"1.2.3.4.5.6.7.8"})
	require.NoError(t, err)
	transferSyntaxEl, err := dicomlib.NewElement(tag.TransferSyntaxUID, []string{uid.ExplicitVRLittleEndian})
	require.NoError(t, err)
	modalityEl, err := dicomlib.NewElement(tag.Modality, []string{modality})
	require.NoError(t, err)

	ds := dicomlib.Dataset{Elements: []*dicomlib.Element{
		sopClassEl,
		sopInstanceEl,
		transferSyntaxEl,
		modalityEl,
	}}

	var buf bytes.Buffer
	require.NoError(t, dicomlib.Write(&buf, ds))
	require.NoError(t, store.Store(context.Background(), key, &buf))
}

// ─── InspectDicomTags tests ───────────────────────────────────────────────────

func TestInspectDicomTags_StudyNotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv, _ := dicomTestServer(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/studies/9.9.9.notexist/dicom-tags", nil)
	req.SetPathValue("studyUID", "9.9.9.notexist")
	rr := httptest.NewRecorder()

	srv.InspectDicomTags(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Contains(t, rr.Body.String(), "study not found")
}

func TestInspectDicomTags_NoFiles(t *testing.T) {
	db := testutil.TestDB(t)
	srv, _ := dicomTestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	req := httptest.NewRequest(http.MethodGet, "/api/studies/"+study.StudyInstanceUID+"/dicom-tags", nil)
	req.SetPathValue("studyUID", study.StudyInstanceUID)
	rr := httptest.NewRecorder()

	srv.InspectDicomTags(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Contains(t, rr.Body.String(), "no DICOM files found")
}

func TestInspectDicomTags_InvalidDicom(t *testing.T) {
	db := testutil.TestDB(t)
	srv, store := dicomTestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	// writeFakeDcm writes non-parseable content — triggers the 422 error path.
	writeFakeDcm(t, store, "dicom/raw/"+study.StudyInstanceUID+"/0.dcm")

	req := httptest.NewRequest(http.MethodGet, "/api/studies/"+study.StudyInstanceUID+"/dicom-tags", nil)
	req.SetPathValue("studyUID", study.StudyInstanceUID)
	rr := httptest.NewRecorder()

	srv.InspectDicomTags(rr, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rr.Code)
	assert.Contains(t, rr.Body.String(), "failed to parse DICOM file")
}

func TestInspectDicomTags_Success(t *testing.T) {
	db := testutil.TestDB(t)
	srv, store := dicomTestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	// Write a real parseable DICOM file.
	writeRealDcm(t, store, "dicom/raw/"+study.StudyInstanceUID+"/0.dcm", "MR")

	req := httptest.NewRequest(http.MethodGet, "/api/studies/"+study.StudyInstanceUID+"/dicom-tags", nil)
	req.SetPathValue("studyUID", study.StudyInstanceUID)
	rr := httptest.NewRecorder()

	srv.InspectDicomTags(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var resp struct {
		StudyUID string `json:"study_uid"`
		File     string `json:"file"`
		TagCount int    `json:"tag_count"`
		Tags     []struct {
			Tag     string `json:"tag"`
			Keyword string `json:"keyword"`
			VR      string `json:"vr"`
			Value   string `json:"value"`
		} `json:"tags"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))

	assert.Equal(t, study.StudyInstanceUID, resp.StudyUID)
	assert.NotEmpty(t, resp.File)
	assert.Greater(t, resp.TagCount, 0)

	// Verify the Modality tag is present with value "MR".
	found := false
	for _, entry := range resp.Tags {
		if entry.Keyword == "Modality" {
			assert.Equal(t, "MR", entry.Value)
			found = true
			break
		}
	}
	assert.True(t, found, "Modality tag should be present in response")
}

func TestInspectDicomTags_SiteScopedOutOfSiteDenied(t *testing.T) {
	db := testutil.TestDB(t)
	srv, _ := dicomTestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	instA := createInstitution(t, db, "sender", "PACS_TAGS_A", true)
	instB := createInstitution(t, db, "sender", "PACS_TAGS_B", true)

	_, err := db.ExecContext(context.Background(), `UPDATE studies SET institution_id = $1 WHERE id = $2`, instA.ID, study.ID)
	require.NoError(t, err)

	researcher := testutil.CreateTestAdminUser(t, db, "dicom-tags-site@test.com", "researcher")
	instBID := instB.ID
	require.NoError(t, model.CreateProjectMember(context.Background(), db, &model.ProjectMember{
		ProjectID:     proj.ID,
		AdminUserID:   researcher.ID,
		Role:          "site_viewer",
		InstitutionID: &instBID,
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/studies/"+study.StudyInstanceUID+"/dicom-tags", nil)
	req.SetPathValue("studyUID", study.StudyInstanceUID)
	req = withResearcherUser(req, researcher.ID, researcher.Email)
	rr := httptest.NewRecorder()

	srv.InspectDicomTags(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Contains(t, rr.Body.String(), "study not found")
}
