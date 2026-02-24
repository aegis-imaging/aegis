package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetUserPreferences_Defaults(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	user := testutil.CreateTestAdminUser(t, db, "prefs-default@test.com", "admin")

	req := httptest.NewRequest(http.MethodGet, "/api/admin-users/"+user.ID+"/preferences", nil)
	req.SetPathValue("id", user.ID)
	rr := httptest.NewRecorder()
	srv.GetUserPreferences(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, "weekly", resp["digest_frequency"])
	assert.Equal(t, []any{}, resp["notify_events"])
}

func TestUpdateUserPreferences_SetAndGet(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	user := testutil.CreateTestAdminUser(t, db, "prefs-set@test.com", "admin")

	body := `{"digest_frequency":"daily","notify_events":["study.stuck","pipeline.failed"]}`
	req := httptest.NewRequest(http.MethodPut, "/api/admin-users/"+user.ID+"/preferences", strings.NewReader(body))
	req.SetPathValue("id", user.ID)
	rr := httptest.NewRecorder()
	srv.UpdateUserPreferences(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, "daily", resp["digest_frequency"])
	events := resp["notify_events"].([]any)
	assert.Len(t, events, 2)

	// GET returns the updated prefs.
	req2 := httptest.NewRequest(http.MethodGet, "/api/admin-users/"+user.ID+"/preferences", nil)
	req2.SetPathValue("id", user.ID)
	rr2 := httptest.NewRecorder()
	srv.GetUserPreferences(rr2, req2)
	assert.Equal(t, http.StatusOK, rr2.Code)

	var resp2 map[string]any
	require.NoError(t, json.NewDecoder(rr2.Body).Decode(&resp2))
	assert.Equal(t, "daily", resp2["digest_frequency"])
}

func TestUpdateUserPreferences_ClearEvents(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	user := testutil.CreateTestAdminUser(t, db, "prefs-clear@test.com", "admin")

	// Set then clear events.
	body1 := `{"digest_frequency":"monthly","notify_events":["study.approved"]}`
	req1 := httptest.NewRequest(http.MethodPut, "/api/admin-users/"+user.ID+"/preferences", strings.NewReader(body1))
	req1.SetPathValue("id", user.ID)
	rr1 := httptest.NewRecorder()
	srv.UpdateUserPreferences(rr1, req1)
	require.Equal(t, http.StatusOK, rr1.Code)

	body2 := `{"digest_frequency":"none","notify_events":[]}`
	req2 := httptest.NewRequest(http.MethodPut, "/api/admin-users/"+user.ID+"/preferences", strings.NewReader(body2))
	req2.SetPathValue("id", user.ID)
	rr2 := httptest.NewRecorder()
	srv.UpdateUserPreferences(rr2, req2)
	assert.Equal(t, http.StatusOK, rr2.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rr2.Body).Decode(&resp))
	assert.Equal(t, "none", resp["digest_frequency"])
	assert.Equal(t, []any{}, resp["notify_events"])
}

func TestUpdateUserPreferences_InvalidFrequency(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	user := testutil.CreateTestAdminUser(t, db, "prefs-bad-freq@test.com", "admin")

	req := httptest.NewRequest(http.MethodPut, "/api/admin-users/"+user.ID+"/preferences",
		strings.NewReader(`{"digest_frequency":"hourly","notify_events":[]}`))
	req.SetPathValue("id", user.ID)
	rr := httptest.NewRecorder()
	srv.UpdateUserPreferences(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestUpdateUserPreferences_InvalidEvent(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	user := testutil.CreateTestAdminUser(t, db, "prefs-bad-ev@test.com", "admin")

	req := httptest.NewRequest(http.MethodPut, "/api/admin-users/"+user.ID+"/preferences",
		strings.NewReader(`{"digest_frequency":"weekly","notify_events":["not.a.real.event"]}`))
	req.SetPathValue("id", user.ID)
	rr := httptest.NewRecorder()
	srv.UpdateUserPreferences(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestGetUserPreferences_UserNotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/admin-users/00000000-0000-0000-0000-000000000000/preferences", nil)
	req.SetPathValue("id", "00000000-0000-0000-0000-000000000000")
	rr := httptest.NewRecorder()
	srv.GetUserPreferences(rr, req)
	assert.Equal(t, http.StatusNotFound, rr.Code)
}
