package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
)

func TestStudyApprovalCRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	// Record sign-off
	req := httptest.NewRequest(http.MethodPost, "/api/studies/"+study.ID+"/approvals", nil)
	req.SetPathValue("id", study.ID)
	w := httptest.NewRecorder()
	srv.RecordStudyApproval(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	// List approvals
	req = httptest.NewRequest(http.MethodGet, "/api/studies/"+study.ID+"/approvals", nil)
	req.SetPathValue("id", study.ID)
	w = httptest.NewRecorder()
	srv.ListStudyApprovals(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "signed_off")
}

func TestStudyApproval_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodPost, "/api/studies/00000000-0000-0000-0000-000000000099/approvals", nil)
	req.SetPathValue("id", "00000000-0000-0000-0000-000000000099")
	w := httptest.NewRecorder()
	srv.RecordStudyApproval(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}
