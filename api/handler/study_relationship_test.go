package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListStudyRelationships_Empty(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	req := httptest.NewRequest(http.MethodGet, "/api/studies/"+study.ID+"/relationships", nil)
	req.SetPathValue("id", study.ID)
	rr := httptest.NewRecorder()
	srv.ListStudyRelationships(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, []any{}, resp["relationships"])
}

func TestCreateStudyRelationship_Success(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	studyA := testutil.CreateTestStudy(t, db, proj.ID)
	studyB := testutil.CreateTestStudy(t, db, proj.ID)

	body := `{"related_study_id":"` + studyB.ID + `","relationship":"follow_up","notes":"6-month scan"}`
	req := httptest.NewRequest(http.MethodPost, "/api/studies/"+studyA.ID+"/relationships", strings.NewReader(body))
	req.SetPathValue("id", studyA.ID)
	rr := httptest.NewRecorder()
	srv.CreateStudyRelationship(rr, req)
	assert.Equal(t, http.StatusCreated, rr.Code)

	var rel map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&rel))
	assert.Equal(t, "follow_up", rel["relationship"])
	assert.Equal(t, studyA.ID, rel["study_id"])
	assert.Equal(t, studyB.ID, rel["related_study_id"])
}

func TestCreateStudyRelationship_DuplicateConflict(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	studyA := testutil.CreateTestStudy(t, db, proj.ID)
	studyB := testutil.CreateTestStudy(t, db, proj.ID)

	body := `{"related_study_id":"` + studyB.ID + `","relationship":"baseline"}`

	// First creation succeeds.
	req1 := httptest.NewRequest(http.MethodPost, "/api/studies/"+studyA.ID+"/relationships", strings.NewReader(body))
	req1.SetPathValue("id", studyA.ID)
	rr1 := httptest.NewRecorder()
	srv.CreateStudyRelationship(rr1, req1)
	require.Equal(t, http.StatusCreated, rr1.Code)

	// Second creation conflicts.
	req2 := httptest.NewRequest(http.MethodPost, "/api/studies/"+studyA.ID+"/relationships", strings.NewReader(body))
	req2.SetPathValue("id", studyA.ID)
	rr2 := httptest.NewRecorder()
	srv.CreateStudyRelationship(rr2, req2)
	assert.Equal(t, http.StatusConflict, rr2.Code)
}

func TestCreateStudyRelationship_SelfLink(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	body := `{"related_study_id":"` + study.ID + `","relationship":"comparison"}`
	req := httptest.NewRequest(http.MethodPost, "/api/studies/"+study.ID+"/relationships", strings.NewReader(body))
	req.SetPathValue("id", study.ID)
	rr := httptest.NewRecorder()
	srv.CreateStudyRelationship(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestCreateStudyRelationship_InvalidType(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	studyA := testutil.CreateTestStudy(t, db, proj.ID)
	studyB := testutil.CreateTestStudy(t, db, proj.ID)

	body := `{"related_study_id":"` + studyB.ID + `","relationship":"twin"}`
	req := httptest.NewRequest(http.MethodPost, "/api/studies/"+studyA.ID+"/relationships", strings.NewReader(body))
	req.SetPathValue("id", studyA.ID)
	rr := httptest.NewRecorder()
	srv.CreateStudyRelationship(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestDeleteStudyRelationship_Success(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	studyA := testutil.CreateTestStudy(t, db, proj.ID)
	studyB := testutil.CreateTestStudy(t, db, proj.ID)

	// Create a relationship first.
	body := `{"related_study_id":"` + studyB.ID + `","relationship":"comparison"}`
	req1 := httptest.NewRequest(http.MethodPost, "/api/studies/"+studyA.ID+"/relationships", strings.NewReader(body))
	req1.SetPathValue("id", studyA.ID)
	rr1 := httptest.NewRecorder()
	srv.CreateStudyRelationship(rr1, req1)
	require.Equal(t, http.StatusCreated, rr1.Code)

	var rel map[string]any
	require.NoError(t, json.NewDecoder(rr1.Body).Decode(&rel))
	relID := rel["id"].(string)

	// Delete it.
	req2 := httptest.NewRequest(http.MethodDelete, "/api/studies/"+studyA.ID+"/relationships/"+relID, nil)
	req2.SetPathValue("id", studyA.ID)
	req2.SetPathValue("relID", relID)
	rr2 := httptest.NewRecorder()
	srv.DeleteStudyRelationship(rr2, req2)
	assert.Equal(t, http.StatusOK, rr2.Code)

	// Now the list should be empty.
	req3 := httptest.NewRequest(http.MethodGet, "/api/studies/"+studyA.ID+"/relationships", nil)
	req3.SetPathValue("id", studyA.ID)
	rr3 := httptest.NewRecorder()
	srv.ListStudyRelationships(rr3, req3)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(rr3.Body).Decode(&resp))
	assert.Equal(t, []any{}, resp["relationships"])
}

func TestListStudyRelationships_ShowsInverseLinks(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	studyA := testutil.CreateTestStudy(t, db, proj.ID)
	studyB := testutil.CreateTestStudy(t, db, proj.ID)

	// Link A → B.
	body := `{"related_study_id":"` + studyB.ID + `","relationship":"baseline"}`
	req1 := httptest.NewRequest(http.MethodPost, "/api/studies/"+studyA.ID+"/relationships", strings.NewReader(body))
	req1.SetPathValue("id", studyA.ID)
	rr1 := httptest.NewRecorder()
	srv.CreateStudyRelationship(rr1, req1)
	require.Equal(t, http.StatusCreated, rr1.Code)

	// Listing from B's perspective should include the inverse link.
	req2 := httptest.NewRequest(http.MethodGet, "/api/studies/"+studyB.ID+"/relationships", nil)
	req2.SetPathValue("id", studyB.ID)
	rr2 := httptest.NewRecorder()
	srv.ListStudyRelationships(rr2, req2)
	assert.Equal(t, http.StatusOK, rr2.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rr2.Body).Decode(&resp))
	rels := resp["relationships"].([]any)
	assert.Len(t, rels, 1, "inverse link should appear in B's relationship list")
}
