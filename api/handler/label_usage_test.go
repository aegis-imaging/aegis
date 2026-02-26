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

func TestGetLabelUsage_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/stats/label-usage", nil)
	w := httptest.NewRecorder()
	srv.GetLabelUsage(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		TotalLabels int           `json:"total_labels"`
		Labels      []interface{} `json:"labels"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.TotalLabels)
	assert.Len(t, resp.Labels, 0)
}

func TestGetLabelUsage_WithLabels(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	s1 := testutil.CreateTestStudy(t, db, proj.ID)
	s2 := testutil.CreateTestStudy(t, db, proj.ID)

	// Add labels
	_, err := db.Exec(`INSERT INTO study_labels (study_id, label) VALUES ($1, 'cohort-A')`, s1.ID)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO study_labels (study_id, label) VALUES ($1, 'cohort-A')`, s2.ID)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO study_labels (study_id, label) VALUES ($1, 'urgent')`, s1.ID)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/stats/label-usage", nil)
	w := httptest.NewRecorder()
	srv.GetLabelUsage(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		TotalLabels int `json:"total_labels"`
		Labels      []struct {
			Label      string `json:"label"`
			Count      int    `json:"count"`
			StudyCount int    `json:"study_count"`
		} `json:"labels"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 2, resp.TotalLabels)
	// cohort-A appears 2 times across 2 studies, urgent appears 1 time across 1 study
	require.Len(t, resp.Labels, 2)
	assert.Equal(t, "cohort-A", resp.Labels[0].Label)
	assert.Equal(t, 2, resp.Labels[0].Count)
	assert.Equal(t, 2, resp.Labels[0].StudyCount)
	assert.Equal(t, "urgent", resp.Labels[1].Label)
	assert.Equal(t, 1, resp.Labels[1].Count)
}

func TestGetLabelUsage_ProjectFilter(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	proj2 := testutil.CreateTestProject(t, db, "Other Project")

	s1 := testutil.CreateTestStudy(t, db, proj.ID)
	s2 := testutil.CreateTestStudy(t, db, proj2.ID)

	_, err := db.Exec(`INSERT INTO study_labels (study_id, label) VALUES ($1, 'shared-label')`, s1.ID)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO study_labels (study_id, label) VALUES ($1, 'shared-label')`, s2.ID)
	require.NoError(t, err)

	// Filter to proj only
	req := httptest.NewRequest(http.MethodGet, "/api/stats/label-usage?project_id="+proj.ID, nil)
	w := httptest.NewRecorder()
	srv.GetLabelUsage(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		ProjectID string `json:"project_id"`
		Labels    []struct {
			Label      string `json:"label"`
			StudyCount int    `json:"study_count"`
		} `json:"labels"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, proj.ID, resp.ProjectID)
	require.Len(t, resp.Labels, 1)
	assert.Equal(t, 1, resp.Labels[0].StudyCount) // only 1 study in proj
}
