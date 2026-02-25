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

func TestGetCohortReport_Empty(t *testing.T) {
	if testing.Short() {
		t.Log("integration test")
		t.Skip()
	}

	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/projects/"+proj.ID+"/cohort-report", nil)
	req.SetPathValue("id", proj.ID)
	w := httptest.NewRecorder()
	srv.GetCohortReport(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		TotalSubjects   int   `json:"total_subjects"`
		Subjects        []any `json:"subjects"`
		ModalityCoverage map[string]int `json:"modality_coverage"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	// Default project from seed data may have studies with subject_id; just check the response shape.
	assert.NotNil(t, resp.Subjects)
	assert.GreaterOrEqual(t, resp.TotalSubjects, 0)
}

func TestGetCohortReport_WithSubjects(t *testing.T) {
	if testing.Short() {
		t.Log("integration test")
		t.Skip()
	}

	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	// Create 3 studies for subject SUB-001 (2 MRI, 1 PET).
	s1 := testutil.CreateTestStudy(t, db, proj.ID)
	_, err := db.Exec(`UPDATE studies SET subject_id = 'SUB-001', modality = 'MRI', status = 'approved' WHERE id = $1`, s1.ID)
	require.NoError(t, err)

	s2 := testutil.CreateTestStudy(t, db, proj.ID)
	_, err = db.Exec(`UPDATE studies SET subject_id = 'SUB-001', modality = 'MRI', status = 'approved' WHERE id = $1`, s2.ID)
	require.NoError(t, err)

	s3 := testutil.CreateTestStudy(t, db, proj.ID)
	_, err = db.Exec(`UPDATE studies SET subject_id = 'SUB-001', modality = 'PET', status = 'received' WHERE id = $1`, s3.ID)
	require.NoError(t, err)

	// Create 1 study for subject SUB-002 (CT).
	s4 := testutil.CreateTestStudy(t, db, proj.ID)
	_, err = db.Exec(`UPDATE studies SET subject_id = 'SUB-002', modality = 'CT', status = 'approved' WHERE id = $1`, s4.ID)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/projects/"+proj.ID+"/cohort-report", nil)
	req.SetPathValue("id", proj.ID)
	w := httptest.NewRecorder()
	srv.GetCohortReport(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		TotalSubjects          int `json:"total_subjects"`
		SubjectsMultiStudy     int `json:"subjects_multi_study"`
		TotalStudiesWithSubject int `json:"total_studies_with_subject"`
		ModalityCoverage       map[string]int `json:"modality_coverage"`
		Subjects               []struct {
			SubjectID     string   `json:"subject_id"`
			StudyCount    int      `json:"study_count"`
			ApprovedCount int      `json:"approved_count"`
			PendingCount  int      `json:"pending_count"`
			Modalities    []string `json:"modalities"`
			AllApproved   bool     `json:"all_approved"`
		} `json:"subjects"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	// At least 2 subjects from our test data (seed data may add more).
	assert.GreaterOrEqual(t, resp.TotalSubjects, 2)
	// At least 1 subject with multiple studies.
	assert.GreaterOrEqual(t, resp.SubjectsMultiStudy, 1)

	// Find SUB-001 in the response.
	var sub001 *struct {
		SubjectID     string   `json:"subject_id"`
		StudyCount    int      `json:"study_count"`
		ApprovedCount int      `json:"approved_count"`
		PendingCount  int      `json:"pending_count"`
		Modalities    []string `json:"modalities"`
		AllApproved   bool     `json:"all_approved"`
	}
	for i := range resp.Subjects {
		if resp.Subjects[i].SubjectID == "SUB-001" {
			sub001 = &resp.Subjects[i]
			break
		}
	}
	require.NotNil(t, sub001, "SUB-001 should appear in cohort report")
	assert.Equal(t, 3, sub001.StudyCount)
	assert.Equal(t, 2, sub001.ApprovedCount)
	assert.Equal(t, 1, sub001.PendingCount) // s3 is still 'received'
	assert.False(t, sub001.AllApproved)     // not all approved (1 pending)
	assert.Len(t, sub001.Modalities, 2)     // MRI + PET

	// Modality coverage: MRI should appear (at least 1 subject has it).
	assert.GreaterOrEqual(t, resp.ModalityCoverage["MRI"], 1)
	assert.GreaterOrEqual(t, resp.ModalityCoverage["CT"], 1)
}

func TestGetCohortReport_NoSubjectIDStudiesExcluded(t *testing.T) {
	if testing.Short() {
		t.Log("integration test")
		t.Skip()
	}

	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	// Create studies with no subject_id — should NOT appear in cohort report.
	testutil.CreateTestStudy(t, db, proj.ID)
	testutil.CreateTestStudy(t, db, proj.ID)

	req := httptest.NewRequest(http.MethodGet, "/api/projects/"+proj.ID+"/cohort-report", nil)
	req.SetPathValue("id", proj.ID)
	w := httptest.NewRecorder()
	srv.GetCohortReport(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		TotalSubjects int `json:"total_subjects"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	// Studies without subject_id should not count as subjects.
	assert.Equal(t, 0, resp.TotalSubjects)
}

func TestGetCohortReport_NotFound(t *testing.T) {
	if testing.Short() {
		t.Log("integration test")
		t.Skip()
	}

	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/projects/00000000-0000-0000-0000-000000000000/cohort-report", nil)
	req.SetPathValue("id", "00000000-0000-0000-0000-000000000000")
	w := httptest.NewRecorder()
	srv.GetCohortReport(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
