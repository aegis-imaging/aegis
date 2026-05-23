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

func TestProjectACL_SetListDelete(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	body := `{"user_email":"alice@test.com","permission":"write"}`
	setReq := httptest.NewRequest(http.MethodPost, "/api/projects/"+proj.ID+"/acl", strings.NewReader(body))
	setReq.SetPathValue("id", proj.ID)
	setRR := httptest.NewRecorder()
	srv.SetProjectACL(setRR, setReq)
	require.Equal(t, http.StatusOK, setRR.Code, setRR.Body.String())

	var entry model.ProjectACLEntry
	require.NoError(t, json.NewDecoder(setRR.Body).Decode(&entry))
	assert.Equal(t, "alice@test.com", entry.UserEmail)
	assert.Equal(t, "write", entry.Permission)
	assert.NotEmpty(t, entry.ID)

	listReq := httptest.NewRequest(http.MethodGet, "/api/projects/"+proj.ID+"/acl", nil)
	listReq.SetPathValue("id", proj.ID)
	listRR := httptest.NewRecorder()
	srv.ListProjectACL(listRR, listReq)
	require.Equal(t, http.StatusOK, listRR.Code)

	var entries []model.ProjectACLEntry
	require.NoError(t, json.NewDecoder(listRR.Body).Decode(&entries))
	require.Len(t, entries, 1)
	assert.Equal(t, "write", entries[0].Permission)

	// Upsert: change permission for same email.
	upsertReq := httptest.NewRequest(http.MethodPost, "/api/projects/"+proj.ID+"/acl",
		strings.NewReader(`{"user_email":"alice@test.com","permission":"admin"}`))
	upsertReq.SetPathValue("id", proj.ID)
	upsertRR := httptest.NewRecorder()
	srv.SetProjectACL(upsertRR, upsertReq)
	require.Equal(t, http.StatusOK, upsertRR.Code)

	delReq := httptest.NewRequest(http.MethodDelete, "/api/projects/"+proj.ID+"/acl/"+entry.ID, nil)
	delReq.SetPathValue("id", proj.ID)
	delReq.SetPathValue("aclID", entry.ID)
	delRR := httptest.NewRecorder()
	srv.DeleteProjectACL(delRR, delReq)
	assert.Equal(t, http.StatusOK, delRR.Code)
}

func TestProjectACL_InvalidPermissionRejected(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+proj.ID+"/acl",
		strings.NewReader(`{"user_email":"x@test.com","permission":"superuser"}`))
	req.SetPathValue("id", proj.ID)
	rr := httptest.NewRecorder()
	srv.SetProjectACL(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestProjectACL_MissingEmailRejected(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+proj.ID+"/acl",
		strings.NewReader(`{"user_email":"","permission":"read"}`))
	req.SetPathValue("id", proj.ID)
	rr := httptest.NewRecorder()
	srv.SetProjectACL(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}
