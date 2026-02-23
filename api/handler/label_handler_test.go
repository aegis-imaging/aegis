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

func TestListStudyLabels_Empty(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	req := httptest.NewRequest("GET", "/api/studies/"+study.ID+"/labels", nil)
	req.SetPathValue("id", study.ID)
	rr := httptest.NewRecorder()
	srv.ListStudyLabels(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var labels []model.StudyLabel
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&labels))
	assert.Empty(t, labels)
}

func TestAddStudyLabel_OK(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	body := `{"label":"cohort-A"}`
	req := httptest.NewRequest("POST", "/api/studies/"+study.ID+"/labels", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", study.ID)
	rr := httptest.NewRecorder()
	srv.AddStudyLabel(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)
	var label model.StudyLabel
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&label))
	assert.Equal(t, "cohort-A", label.Label)
	assert.Equal(t, study.ID, label.StudyID)
}

func TestAddStudyLabel_StudyNotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body := `{"label":"cohort-A"}`
	req := httptest.NewRequest("POST", "/api/studies/00000000-0000-0000-0000-000000000000/labels", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "00000000-0000-0000-0000-000000000000")
	rr := httptest.NewRecorder()
	srv.AddStudyLabel(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestAddStudyLabel_EmptyLabel(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	req := httptest.NewRequest("POST", "/api/studies/"+study.ID+"/labels", bytes.NewBufferString(`{"label":"  "}`))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", study.ID)
	rr := httptest.NewRecorder()
	srv.AddStudyLabel(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestDeleteStudyLabel_OK(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	label, err := model.AddStudyLabel(context.Background(), db, study.ID, "to-delete", "test@local")
	require.NoError(t, err)

	req := httptest.NewRequest("DELETE", "/api/studies/"+study.ID+"/labels/"+label.ID, nil)
	req.SetPathValue("id", study.ID)
	req.SetPathValue("labelID", label.ID)
	rr := httptest.NewRecorder()
	srv.DeleteStudyLabel(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	// Verify deleted
	remaining, err := model.ListStudyLabels(context.Background(), db, study.ID)
	require.NoError(t, err)
	assert.Empty(t, remaining)
}

func TestBulkLabelStudies_Add(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	s1 := testutil.CreateTestStudy(t, db, proj.ID)
	s2 := testutil.CreateTestStudy(t, db, proj.ID)

	body, _ := json.Marshal(map[string]any{
		"study_ids": []string{s1.ID, s2.ID},
		"label":    "batch-label",
		"action":   "add",
	})
	req := httptest.NewRequest("POST", "/api/studies/bulk-label", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.BulkLabelStudies(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, float64(2), resp["total"])
}

func TestBulkLabelStudies_Remove(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)
	_, err := model.AddStudyLabel(context.Background(), db, study.ID, "to-remove", "test@local")
	require.NoError(t, err)

	body, _ := json.Marshal(map[string]any{
		"study_ids": []string{study.ID},
		"label":    "to-remove",
		"action":   "remove",
	})
	req := httptest.NewRequest("POST", "/api/studies/bulk-label", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.BulkLabelStudies(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, float64(1), resp["removed"])
}

func TestBulkLabelStudies_InvalidAction(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	body, _ := json.Marshal(map[string]any{
		"study_ids": []string{study.ID},
		"label":    "x",
		"action":   "delete",
	})
	req := httptest.NewRequest("POST", "/api/studies/bulk-label", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.BulkLabelStudies(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}
