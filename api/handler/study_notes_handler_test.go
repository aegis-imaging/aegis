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

func TestListStudyNotes_NotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest("GET", "/api/studies/00000000-0000-0000-0000-000000000000/notes", nil)
	req.SetPathValue("id", "00000000-0000-0000-0000-000000000000")
	rr := httptest.NewRecorder()
	srv.ListStudyNotes(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestListStudyNotes_Empty(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	req := httptest.NewRequest("GET", "/api/studies/"+study.ID+"/notes", nil)
	req.SetPathValue("id", study.ID)
	rr := httptest.NewRecorder()
	srv.ListStudyNotes(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp struct {
		StudyID string        `json:"study_id"`
		Notes   []interface{} `json:"notes"`
		Total   int           `json:"total"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, study.ID, resp.StudyID)
	assert.Equal(t, 0, resp.Total)
	assert.Len(t, resp.Notes, 0)
}

func TestListStudyNotes_WithNotes(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	// Add notes via audit entries
	model.CreateAuditEntry(context.Background(), db, "study.note", "admin@test.com", "study", study.ID, "127.0.0.1", map[string]any{"note": "First note"})
	model.CreateAuditEntry(context.Background(), db, "study.note", "admin@test.com", "study", study.ID, "127.0.0.1", map[string]any{"note": "Second note"})

	req := httptest.NewRequest("GET", "/api/studies/"+study.ID+"/notes", nil)
	req.SetPathValue("id", study.ID)
	rr := httptest.NewRecorder()
	srv.ListStudyNotes(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp struct {
		Total int `json:"total"`
		Notes []struct {
			Actor string `json:"actor"`
			Note  string `json:"note"`
		} `json:"notes"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, 2, resp.Total)
	require.Len(t, resp.Notes, 2)
	// Newest first
	assert.Equal(t, "Second note", resp.Notes[0].Note)
	assert.Equal(t, "First note", resp.Notes[1].Note)
	assert.Equal(t, "admin@test.com", resp.Notes[0].Actor)
}
