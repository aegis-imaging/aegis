package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
)

func TestCompareStudies(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	studyA := testutil.CreateTestStudy(t, db, proj.ID)
	studyB := testutil.CreateTestStudy(t, db, proj.ID)

	req := httptest.NewRequest(http.MethodGet, "/api/studies/compare?a="+studyA.ID+"&b="+studyB.ID, nil)
	w := httptest.NewRecorder()
	srv.CompareStudies(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "diff")
	assert.Contains(t, w.Body.String(), "modality")
}

func TestCompareStudies_MissingParams(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/studies/compare?a=123", nil)
	w := httptest.NewRecorder()
	srv.CompareStudies(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
