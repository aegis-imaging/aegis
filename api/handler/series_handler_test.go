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

func TestListStudySeries_Empty(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	req := httptest.NewRequest("GET", "/api/studies/"+study.ID+"/series", nil)
	req.SetPathValue("id", study.ID)
	rr := httptest.NewRecorder()
	srv.ListStudySeries(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var result map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.Equal(t, study.ID, result["study_id"])
	assert.EqualValues(t, 0, result["total"])
	series, ok := result["series"].([]any)
	require.True(t, ok)
	assert.Len(t, series, 0)
}

func TestListStudySeries_ReturnsRows(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	s1 := &model.StudySeries{
		StudyID:           study.ID,
		SeriesInstanceUID: "1.2.3.4.100",
		SeriesDescription: "T1w MPRAGE",
		Modality:          "MR",
		BodyPart:          "HEAD",
		InstanceCount:     176,
	}
	s2 := &model.StudySeries{
		StudyID:           study.ID,
		SeriesInstanceUID: "1.2.3.4.200",
		SeriesDescription: "FLAIR",
		Modality:          "MR",
		BodyPart:          "HEAD",
		InstanceCount:     48,
	}
	require.NoError(t, model.UpsertStudySeries(t.Context(), db, s1))
	require.NoError(t, model.UpsertStudySeries(t.Context(), db, s2))

	req := httptest.NewRequest("GET", "/api/studies/"+study.ID+"/series", nil)
	req.SetPathValue("id", study.ID)
	rr := httptest.NewRecorder()
	srv.ListStudySeries(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var result map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.EqualValues(t, 2, result["total"])
	series := result["series"].([]any)
	require.Len(t, series, 2)

	// Rows ordered by series_instance_uid
	row0 := series[0].(map[string]any)
	assert.Equal(t, "1.2.3.4.100", row0["series_instance_uid"])
	assert.Equal(t, "T1w MPRAGE", row0["series_description"])
	assert.EqualValues(t, 176, row0["instance_count"])
}

func TestListStudySeries_Upsert_Idempotent(t *testing.T) {
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	s := &model.StudySeries{
		StudyID:           study.ID,
		SeriesInstanceUID: "1.2.3.9.999",
		SeriesDescription: "Original",
		InstanceCount:     10,
	}
	require.NoError(t, model.UpsertStudySeries(t.Context(), db, s))

	// Upsert with updated description and count
	s.SeriesDescription = "Updated"
	s.InstanceCount = 20
	require.NoError(t, model.UpsertStudySeries(t.Context(), db, s))

	rows, err := model.ListStudySeries(t.Context(), db, study.ID)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "Updated", rows[0].SeriesDescription)
	assert.Equal(t, 20, rows[0].InstanceCount)
}

func TestListStudySeries_NotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest("GET", "/api/studies/00000000-0000-0000-0000-000000000000/series", nil)
	req.SetPathValue("id", "00000000-0000-0000-0000-000000000000")
	rr := httptest.NewRecorder()
	srv.ListStudySeries(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestListStudySeries_SiteScopedOutOfSiteDenied(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	instA := createInstitution(t, db, "sender", "PACS_SERIES_A", true)
	instB := createInstitution(t, db, "sender", "PACS_SERIES_B", true)

	_, err := db.ExecContext(context.Background(), `UPDATE studies SET institution_id = $1 WHERE id = $2`, instA.ID, study.ID)
	require.NoError(t, err)

	researcher := testutil.CreateTestAdminUser(t, db, "series-site@test.com", "researcher")
	instBID := instB.ID
	require.NoError(t, model.CreateProjectMember(context.Background(), db, &model.ProjectMember{
		ProjectID:     proj.ID,
		AdminUserID:   researcher.ID,
		Role:          "site_coordinator",
		InstitutionID: &instBID,
	}))

	req := httptest.NewRequest("GET", "/api/studies/"+study.ID+"/series", nil)
	req.SetPathValue("id", study.ID)
	req = withResearcherUser(req, researcher.ID, researcher.Email)
	rr := httptest.NewRecorder()

	srv.ListStudySeries(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Contains(t, rr.Body.String(), "study not found")
}
