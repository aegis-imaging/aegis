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

func TestReviewChecklist_CRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	// Create checklist item
	body := `{"label":"Verify modality","sort_order":1,"required":true}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+proj.ID+"/review-checklist", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("projectID", proj.ID)
	w := httptest.NewRecorder()
	srv.CreateReviewChecklistItem(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	var item model.ReviewChecklistItem
	require.NoError(t, json.NewDecoder(w.Body).Decode(&item))
	assert.Equal(t, "Verify modality", item.Label)
	assert.True(t, item.Required)

	// List items
	req = httptest.NewRequest(http.MethodGet, "/api/projects/"+proj.ID+"/review-checklist", nil)
	req.SetPathValue("projectID", proj.ID)
	w = httptest.NewRecorder()
	srv.ListReviewChecklist(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var items []model.ReviewChecklistItem
	require.NoError(t, json.NewDecoder(w.Body).Decode(&items))
	assert.Len(t, items, 1)

	// Delete item
	req = httptest.NewRequest(http.MethodDelete, "/api/review-checklist/"+item.ID, nil)
	req.SetPathValue("id", item.ID)
	w = httptest.NewRecorder()
	srv.DeleteReviewChecklistItem(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestReviewChecklist_EmptyLabel(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	body := `{"label":"","sort_order":1,"required":true}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+proj.ID+"/review-checklist", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("projectID", proj.ID)
	w := httptest.NewRecorder()
	srv.CreateReviewChecklistItem(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestStudyChecklistResponse(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	// Create a checklist item
	body := `{"label":"Check PHI","sort_order":0,"required":true}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+proj.ID+"/review-checklist", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("projectID", proj.ID)
	w := httptest.NewRecorder()
	srv.CreateReviewChecklistItem(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var item model.ReviewChecklistItem
	require.NoError(t, json.NewDecoder(w.Body).Decode(&item))

	// Upsert checklist response
	body = `{"checklist_item_id":"` + item.ID + `","checked":true}`
	req = httptest.NewRequest(http.MethodPost, "/api/studies/"+study.ID+"/checklist", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", study.ID)
	w = httptest.NewRecorder()
	srv.UpsertStudyChecklistResponse(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// List responses
	req = httptest.NewRequest(http.MethodGet, "/api/studies/"+study.ID+"/checklist", nil)
	req.SetPathValue("id", study.ID)
	w = httptest.NewRecorder()
	srv.ListStudyChecklistResponses(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var responses []model.StudyChecklistResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&responses))
	assert.Len(t, responses, 1)
	assert.True(t, responses[0].Checked)
}
