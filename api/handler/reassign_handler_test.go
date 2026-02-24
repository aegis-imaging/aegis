package handler_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestProject creates a second project directly via SQL for reassignment tests.
func createSecondProject(t *testing.T, db *sql.DB) *model.Project {
	t.Helper()
	p := &model.Project{}
	err := db.QueryRowContext(context.Background(), `
		INSERT INTO projects (name, slug, description)
		VALUES ('Second Project', 'second-project', 'test') RETURNING id, name, slug, description, created_at`,
	).Scan(&p.ID, &p.Name, &p.Slug, &p.Description, &p.CreatedAt)
	require.NoError(t, err)
	return p
}

func TestReassignStudy_OK(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj1 := testutil.SeedProject(t, db)
	proj2 := createSecondProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj1.ID)

	body, _ := json.Marshal(map[string]string{"project_id": proj2.ID})
	req := httptest.NewRequest("PUT", "/api/studies/"+study.ID+"/project", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", study.ID)
	rr := httptest.NewRecorder()
	srv.ReassignStudy(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp model.Study
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, proj2.ID, resp.ProjectID)
}

func TestReassignStudy_SameProject(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	body, _ := json.Marshal(map[string]string{"project_id": proj.ID})
	req := httptest.NewRequest("PUT", "/api/studies/"+study.ID+"/project", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", study.ID)
	rr := httptest.NewRecorder()
	srv.ReassignStudy(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestReassignStudy_StudyNotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	body, _ := json.Marshal(map[string]string{"project_id": proj.ID})
	req := httptest.NewRequest("PUT", "/api/studies/00000000-0000-0000-0000-000000000000/project", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "00000000-0000-0000-0000-000000000000")
	rr := httptest.NewRecorder()
	srv.ReassignStudy(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestReassignStudy_TargetProjectNotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	body, _ := json.Marshal(map[string]string{"project_id": "00000000-0000-0000-0000-000000000000"})
	req := httptest.NewRequest("PUT", "/api/studies/"+study.ID+"/project", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", study.ID)
	rr := httptest.NewRecorder()
	srv.ReassignStudy(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}
