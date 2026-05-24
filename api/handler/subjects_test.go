package handler_test

import (
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

func TestListProjectSubjects_EmptyProject(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/projects/"+proj.ID+"/subjects", nil)
	req.SetPathValue("id", proj.ID)
	rr := httptest.NewRecorder()
	srv.ListProjectSubjects(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var body struct {
		ProjectID string                  `json:"project_id"`
		Subjects  []model.SubjectAggregate `json:"subjects"`
		Total     int                     `json:"total"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
	assert.Equal(t, proj.ID, body.ProjectID)
	assert.Equal(t, 0, body.Total)
	assert.Empty(t, body.Subjects)
}

func TestListProjectSubjects_GroupsBySubjectID(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	// Two subjects, three studies total.
	s1 := testutil.CreateTestStudy(t, db, proj.ID)
	s2 := testutil.CreateTestStudy(t, db, proj.ID)
	s3 := testutil.CreateTestStudy(t, db, proj.ID)
	subj1 := "SUBJ-001"
	subj2 := "SUBJ-002"
	require.NoError(t, model.UpdateStudySubjectID(context.Background(), db, s1.ID, &subj1))
	require.NoError(t, model.UpdateStudySubjectID(context.Background(), db, s2.ID, &subj1))
	require.NoError(t, model.UpdateStudySubjectID(context.Background(), db, s3.ID, &subj2))

	req := httptest.NewRequest(http.MethodGet, "/api/projects/"+proj.ID+"/subjects", nil)
	req.SetPathValue("id", proj.ID)
	rr := httptest.NewRecorder()
	srv.ListProjectSubjects(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var body struct {
		Subjects []model.SubjectAggregate `json:"subjects"`
		Total    int                     `json:"total"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
	assert.Equal(t, 2, body.Total)
	require.Len(t, body.Subjects, 2)
	// Order is ASC by subject_id, so SUBJ-001 first.
	assert.Equal(t, "SUBJ-001", body.Subjects[0].SubjectID)
	assert.Equal(t, 2, body.Subjects[0].StudyCount)
	assert.Equal(t, "SUBJ-002", body.Subjects[1].SubjectID)
	assert.Equal(t, 1, body.Subjects[1].StudyCount)
}

func TestListProjectSubjects_MergesDemographics(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	study := testutil.CreateTestStudy(t, db, proj.ID)
	subj := "SUBJ-DEMO"
	require.NoError(t, model.UpdateStudySubjectID(context.Background(), db, study.ID, &subj))

	age := 42
	demo := &model.SubjectDemographics{
		SubjectID: subj,
		ProjectID: proj.ID,
		Sex:       "F",
		AgeAtScan: &age,
		Diagnosis: "Control",
	}
	require.NoError(t, model.UpsertSubjectDemographics(context.Background(), db, demo))

	req := httptest.NewRequest(http.MethodGet, "/api/projects/"+proj.ID+"/subjects", nil)
	req.SetPathValue("id", proj.ID)
	rr := httptest.NewRecorder()
	srv.ListProjectSubjects(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var body struct {
		Subjects []model.SubjectAggregate `json:"subjects"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
	require.Len(t, body.Subjects, 1)
	require.NotNil(t, body.Subjects[0].Demographics)
	assert.Equal(t, "F", body.Subjects[0].Demographics.Sex)
	assert.Equal(t, "Control", body.Subjects[0].Demographics.Diagnosis)
}

func TestGetProjectSubject_NotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/projects/"+proj.ID+"/subjects/SUBJ-NOPE", nil)
	req.SetPathValue("id", proj.ID)
	req.SetPathValue("subjectID", "SUBJ-NOPE")
	rr := httptest.NewRecorder()
	srv.GetProjectSubject(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestGetProjectSubject_ReturnsAggregateAndStudies(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	s1 := testutil.CreateTestStudy(t, db, proj.ID)
	s2 := testutil.CreateTestStudy(t, db, proj.ID)
	subj := "SUBJ-DRILL"
	require.NoError(t, model.UpdateStudySubjectID(context.Background(), db, s1.ID, &subj))
	require.NoError(t, model.UpdateStudySubjectID(context.Background(), db, s2.ID, &subj))

	req := httptest.NewRequest(http.MethodGet, "/api/projects/"+proj.ID+"/subjects/"+subj, nil)
	req.SetPathValue("id", proj.ID)
	req.SetPathValue("subjectID", subj)
	rr := httptest.NewRecorder()
	srv.GetProjectSubject(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var body struct {
		SubjectID  string        `json:"subject_id"`
		StudyCount int           `json:"study_count"`
		Studies    []model.Study `json:"studies"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
	assert.Equal(t, subj, body.SubjectID)
	assert.Equal(t, 2, body.StudyCount)
	assert.Len(t, body.Studies, 2)
}

func TestSearch_EmptyQuery(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	_ = testutil.SeedProject(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/search?q=", nil)
	rr := httptest.NewRecorder()
	srv.Search(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var body struct {
		Query    string `json:"query"`
		Projects []any  `json:"projects"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
	assert.Equal(t, "", body.Query)
	assert.Empty(t, body.Projects)
}

func TestSearch_FindsBySubjectID(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	study := testutil.CreateTestStudy(t, db, proj.ID)
	subj := "SUBJ-ALPHA-123"
	require.NoError(t, model.UpdateStudySubjectID(context.Background(), db, study.ID, &subj))

	req := httptest.NewRequest(http.MethodGet, "/api/search?q=ALPHA", nil)
	rr := httptest.NewRecorder()
	srv.Search(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var body struct {
		Subjects []struct {
			SubjectID string `json:"subject_id"`
		} `json:"subjects"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
	require.Len(t, body.Subjects, 1)
	assert.Equal(t, subj, body.Subjects[0].SubjectID)
}
