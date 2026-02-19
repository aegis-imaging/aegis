package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/msenjem/aegis/api/model"
	"github.com/msenjem/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateAdminUser_Handler(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body, _ := json.Marshal(map[string]any{"email": "new@test.com", "name": "New User", "role": "admin"})
	req := httptest.NewRequest("POST", "/api/admin-users", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	srv.CreateAdminUser(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)
	var u model.AdminUser
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&u))
	assert.Equal(t, "new@test.com", u.Email)
	assert.Equal(t, "admin", u.Role)
	assert.True(t, u.Enabled)
}

func TestCreateAdminUser_MissingEmail(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body, _ := json.Marshal(map[string]any{"name": "No Email"})
	req := httptest.NewRequest("POST", "/api/admin-users", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	srv.CreateAdminUser(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestCreateAdminUser_DefaultRole(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body, _ := json.Marshal(map[string]any{"email": "default-role@test.com", "name": "Test"})
	req := httptest.NewRequest("POST", "/api/admin-users", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	srv.CreateAdminUser(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)
	var u model.AdminUser
	json.NewDecoder(rr.Body).Decode(&u)
	assert.Equal(t, "admin", u.Role, "role should default to admin")
}

func TestListAdminUsers_Handler(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest("GET", "/api/admin-users", nil)
	rr := httptest.NewRecorder()
	srv.ListAdminUsers(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var users []model.AdminUser
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&users))
	// Empty array, not null
	assert.NotNil(t, users)
}

func TestDeleteAdminUser_Handler(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	u := testutil.CreateTestAdminUser(t, db, "del@test.com", "admin")

	req := httptest.NewRequest("DELETE", "/api/admin-users/"+u.ID, nil)
	req.SetPathValue("id", u.ID)
	rr := httptest.NewRecorder()
	srv.DeleteAdminUser(rr, req)

	assert.Equal(t, http.StatusNoContent, rr.Code)
}

func TestDeleteAdminUser_NotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest("DELETE", "/api/admin-users/nonexistent", nil)
	req.SetPathValue("id", "nonexistent")
	rr := httptest.NewRecorder()
	srv.DeleteAdminUser(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}
