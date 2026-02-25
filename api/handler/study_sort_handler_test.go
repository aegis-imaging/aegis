package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestListStudies_Sort verifies that sort_by and sort_dir query parameters
// are accepted and produce a 200 response (correctness of ordering is
// verified in the model-level tests).
func TestListStudies_SortByStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	testutil.CreateTestStudy(t, db, proj.ID)

	req := httptest.NewRequest("GET", "/api/studies?sort_by=status&sort_dir=asc", nil)
	rr := httptest.NewRecorder()
	srv.ListStudies(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var result struct {
		Studies []model.Study `json:"studies"`
		Total   int           `json:"total"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.GreaterOrEqual(t, result.Total, 1)
}

func TestListStudies_SortByInstanceCount(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	testutil.CreateTestStudy(t, db, proj.ID)
	testutil.CreateTestStudy(t, db, proj.ID)

	req := httptest.NewRequest("GET", "/api/studies?sort_by=instance_count&sort_dir=desc", nil)
	rr := httptest.NewRecorder()
	srv.ListStudies(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var result struct {
		Studies []model.Study `json:"studies"`
		Total   int           `json:"total"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.GreaterOrEqual(t, result.Total, 2)
}

func TestListStudies_InvalidSortBy_FallsBackToDefault(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	testutil.CreateTestStudy(t, db, proj.ID)

	// An unknown sort_by value should fall back to created_at DESC without error.
	req := httptest.NewRequest("GET", "/api/studies?sort_by=unknown_column&sort_dir=asc", nil)
	rr := httptest.NewRecorder()
	srv.ListStudies(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestListStudies_SortByCreatedAtAsc(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	testutil.CreateTestStudy(t, db, proj.ID)
	testutil.CreateTestStudy(t, db, proj.ID)

	req := httptest.NewRequest("GET", "/api/studies?sort_by=created_at&sort_dir=asc", nil)
	rr := httptest.NewRecorder()
	srv.ListStudies(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var result struct {
		Studies []model.Study `json:"studies"`
		Total   int           `json:"total"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	// With asc ordering, earliest study should be first.
	require.GreaterOrEqual(t, len(result.Studies), 2)
	assert.True(t, !result.Studies[0].CreatedAt.After(result.Studies[1].CreatedAt),
		"asc sort: first study should not be newer than second")
}
