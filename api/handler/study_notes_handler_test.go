package handler_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
)

func TestAddStudyNote_OK(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	body := `{"note":"This study needs manual review"}`
	req := httptest.NewRequest("POST", "/api/studies/"+study.ID+"/notes", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", study.ID)
	rr := httptest.NewRecorder()
	srv.AddStudyNote(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestAddStudyNote_StudyNotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body := `{"note":"some note"}`
	req := httptest.NewRequest("POST", "/api/studies/00000000-0000-0000-0000-000000000000/notes", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "00000000-0000-0000-0000-000000000000")
	rr := httptest.NewRecorder()
	srv.AddStudyNote(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestAddStudyNote_EmptyNote(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	req := httptest.NewRequest("POST", "/api/studies/"+study.ID+"/notes", bytes.NewBufferString(`{"note":""}`))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", study.ID)
	rr := httptest.NewRecorder()
	srv.AddStudyNote(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestAddStudyNote_TooLong(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	note := make([]byte, 2001)
	for i := range note {
		note[i] = 'x'
	}
	body := `{"note":"` + string(note) + `"}`
	req := httptest.NewRequest("POST", "/api/studies/"+study.ID+"/notes", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", study.ID)
	rr := httptest.NewRecorder()
	srv.AddStudyNote(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}
