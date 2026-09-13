package handler_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
)

func TestTransferStudy(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	proj2 := testutil.CreateTestProject(t, db, "transfer-target")
	study := testutil.CreateTestStudy(t, db, proj.ID)

	// Transfer study
	body := `{"to_project_id":"` + proj2.ID + `","reason":"reassignment"}`
	req := httptest.NewRequest(http.MethodPost, "/api/studies/"+study.ID+"/transfer", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", study.ID)
	w := httptest.NewRecorder()
	srv.TransferStudy(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// List transfers
	req = httptest.NewRequest(http.MethodGet, "/api/studies/"+study.ID+"/transfers", nil)
	req.SetPathValue("id", study.ID)
	w = httptest.NewRecorder()
	srv.ListStudyTransfers(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), proj2.ID)
}

func TestTransferStudy_MissingProjectID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/api/studies/"+study.ID+"/transfer", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", study.ID)
	w := httptest.NewRecorder()
	srv.TransferStudy(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTransferStudy_SameProject(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	body := `{"to_project_id":"` + proj.ID + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/studies/"+study.ID+"/transfer", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", study.ID)
	w := httptest.NewRecorder()
	srv.TransferStudy(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
