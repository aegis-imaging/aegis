package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddAndListStudyTags(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	addReq := httptest.NewRequest(http.MethodPost, "/api/studies/"+study.ID+"/tags", strings.NewReader(`{"tag":"oncology"}`))
	addReq.SetPathValue("id", study.ID)
	addRR := httptest.NewRecorder()
	srv.AddStudyTag(addRR, addReq)
	require.Equal(t, http.StatusCreated, addRR.Code, addRR.Body.String())

	var tag model.StudyTag
	require.NoError(t, json.NewDecoder(addRR.Body).Decode(&tag))
	assert.Equal(t, "oncology", tag.Tag)
	assert.Equal(t, study.ID, tag.StudyID)

	listReq := httptest.NewRequest(http.MethodGet, "/api/studies/"+study.ID+"/tags", nil)
	listReq.SetPathValue("id", study.ID)
	listRR := httptest.NewRecorder()
	srv.ListStudyTags(listRR, listReq)
	assert.Equal(t, http.StatusOK, listRR.Code)

	var tags []model.StudyTag
	require.NoError(t, json.NewDecoder(listRR.Body).Decode(&tags))
	require.Len(t, tags, 1)
	assert.Equal(t, "oncology", tags[0].Tag)
}

func TestAddStudyTag_EmptyTagRejected(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	req := httptest.NewRequest(http.MethodPost, "/api/studies/"+study.ID+"/tags", strings.NewReader(`{"tag":"  "}`))
	req.SetPathValue("id", study.ID)
	rr := httptest.NewRecorder()
	srv.AddStudyTag(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestAddStudyTag_StudyNotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodPost, "/api/studies/missing/tags", strings.NewReader(`{"tag":"x"}`))
	req.SetPathValue("id", "00000000-0000-0000-0000-000000000000")
	rr := httptest.NewRecorder()
	srv.AddStudyTag(rr, req)
	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestProjectTagTaxonomy(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	s1 := testutil.CreateTestStudy(t, db, proj.ID)
	s2 := testutil.CreateTestStudy(t, db, proj.ID)

	for _, sid := range []string{s1.ID, s2.ID} {
		r := httptest.NewRequest(http.MethodPost, "/api/studies/"+sid+"/tags", strings.NewReader(`{"tag":"shared"}`))
		r.SetPathValue("id", sid)
		rr := httptest.NewRecorder()
		srv.AddStudyTag(rr, r)
		require.Equal(t, http.StatusCreated, rr.Code)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/projects/"+proj.ID+"/tag-taxonomy", nil)
	req.SetPathValue("id", proj.ID)
	rr := httptest.NewRecorder()
	srv.GetProjectTagTaxonomy(rr, req)
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

	var resp struct {
		Tags  []model.TagCount `json:"tags"`
		Total int              `json:"total"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	require.GreaterOrEqual(t, resp.Total, 1)
	found := false
	for _, t := range resp.Tags {
		if t.Tag == "shared" && t.Count == 2 {
			found = true
		}
	}
	assert.True(t, found, "expected shared tag with count 2 in taxonomy: %+v", resp.Tags)
}
