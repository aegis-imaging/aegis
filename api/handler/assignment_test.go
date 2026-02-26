package handler_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
)

func TestAssignStudy(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)
	user := testutil.CreateTestAdminUser(t, db, "assign@test.com", "admin")

	// Assign study
	body := `{"user_id":"` + user.ID + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/studies/"+study.ID+"/assign", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", study.ID)
	w := httptest.NewRecorder()
	srv.AssignStudy(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Unassign study
	req = httptest.NewRequest(http.MethodDelete, "/api/studies/"+study.ID+"/assign", nil)
	req.SetPathValue("id", study.ID)
	w = httptest.NewRecorder()
	srv.UnassignStudy(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAssignStudy_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body := `{"user_id":"00000000-0000-0000-0000-000000000001"}`
	req := httptest.NewRequest(http.MethodPost, "/api/studies/00000000-0000-0000-0000-000000000099/assign", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "00000000-0000-0000-0000-000000000099")
	w := httptest.NewRecorder()
	srv.AssignStudy(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAssignStudy_MissingUserID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/api/studies/"+study.ID+"/assign", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", study.ID)
	w := httptest.NewRecorder()
	srv.AssignStudy(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
