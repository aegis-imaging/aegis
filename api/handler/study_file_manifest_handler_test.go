package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStudyFileManifest_OK(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	req := httptest.NewRequest(http.MethodGet, "/api/studies/"+study.ID+"/files", nil)
	req.SetPathValue("id", study.ID)
	rr := httptest.NewRecorder()
	srv.GetStudyFileManifest(rr, req)
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

	var resp struct {
		StudyID       string `json:"study_id"`
		InstanceCount int    `json:"instance_count"`
		DicomStore    string `json:"dicom_store"`
		Files         []struct {
			Index    int    `json:"index"`
			Filename string `json:"filename"`
			Store    string `json:"store"`
		} `json:"files"`
		Total int `json:"total"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, study.ID, resp.StudyID)
	assert.Equal(t, study.InstanceCount, resp.InstanceCount)
	assert.Equal(t, "raw", resp.DicomStore)
	assert.Len(t, resp.Files, study.InstanceCount)
	assert.Equal(t, 0, resp.Files[0].Index)
	assert.Contains(t, resp.Files[0].Filename, study.StudyInstanceUID)
}

func TestStudyFileManifest_LimitParam(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	req := httptest.NewRequest(http.MethodGet, "/api/studies/"+study.ID+"/files?limit=3", nil)
	req.SetPathValue("id", study.ID)
	rr := httptest.NewRecorder()
	srv.GetStudyFileManifest(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	var resp struct {
		Files []any `json:"files"`
		Total int   `json:"total"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Len(t, resp.Files, 3)
	assert.Equal(t, 3, resp.Total)
}

func TestStudyFileManifest_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/studies/missing/files", nil)
	req.SetPathValue("id", "00000000-0000-0000-0000-000000000000")
	rr := httptest.NewRecorder()
	srv.GetStudyFileManifest(rr, req)
	assert.Equal(t, http.StatusNotFound, rr.Code)
}
