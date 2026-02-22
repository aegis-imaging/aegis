package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListSubjects_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest("GET", "/api/subjects", nil)
	rr := httptest.NewRecorder()
	srv.ListSubjects(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var subjects []model.SubjectSummary
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&subjects))
	assert.NotNil(t, subjects)
}

func TestSetStudySubject_SetAndClear(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	// Set a subject ID
	body, _ := json.Marshal(map[string]string{"subject_id": "SUBJ-001"})
	req := httptest.NewRequest("PUT", "/api/studies/"+study.ID+"/subject", bytes.NewReader(body))
	req.SetPathValue("id", study.ID)
	rr := httptest.NewRecorder()
	srv.SetStudySubject(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var s model.Study
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&s))
	require.NotNil(t, s.SubjectID)
	assert.Equal(t, "SUBJ-001", *s.SubjectID)

	// Clear the subject ID (empty string)
	body2, _ := json.Marshal(map[string]string{"subject_id": ""})
	req2 := httptest.NewRequest("PUT", "/api/studies/"+study.ID+"/subject", bytes.NewReader(body2))
	req2.SetPathValue("id", study.ID)
	rr2 := httptest.NewRecorder()
	srv.SetStudySubject(rr2, req2)

	assert.Equal(t, http.StatusOK, rr2.Code)
	var s2 model.Study
	require.NoError(t, json.NewDecoder(rr2.Body).Decode(&s2))
	assert.Nil(t, s2.SubjectID)
}

func TestSetStudySubject_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body, _ := json.Marshal(map[string]string{"subject_id": "SUBJ-001"})
	req := httptest.NewRequest("PUT", "/api/studies/nonexistent/subject", bytes.NewReader(body))
	req.SetPathValue("id", "nonexistent")
	rr := httptest.NewRecorder()
	srv.SetStudySubject(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestListSubjects_WithSubjectID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	// Assign a subject ID via model
	sid := "SUBJ-TEST"
	require.NoError(t, model.UpdateStudySubjectID(context.Background(), db, study.ID, &sid))

	req := httptest.NewRequest("GET", "/api/subjects?project_id="+proj.ID, nil)
	rr := httptest.NewRecorder()
	srv.ListSubjects(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var subjects []model.SubjectSummary
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&subjects))
	require.Len(t, subjects, 1)
	assert.Equal(t, "SUBJ-TEST", subjects[0].SubjectID)
	assert.Equal(t, 1, subjects[0].StudyCount)
}
