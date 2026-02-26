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

func TestAutoShareRules_CRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	// Create auto-share rule
	body := `{"recipient_email":"auto@test.com","expiry_hours":48,"note":"test rule"}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+proj.ID+"/auto-share-rules", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("projectID", proj.ID)
	w := httptest.NewRecorder()
	srv.CreateAutoShareRule(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	var rule model.AutoShareRule
	require.NoError(t, json.NewDecoder(w.Body).Decode(&rule))
	assert.Equal(t, "auto@test.com", rule.RecipientEmail)
	assert.Equal(t, 48, rule.ExpiryHours)

	// List rules
	req = httptest.NewRequest(http.MethodGet, "/api/projects/"+proj.ID+"/auto-share-rules", nil)
	req.SetPathValue("projectID", proj.ID)
	w = httptest.NewRecorder()
	srv.ListAutoShareRules(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var rules []model.AutoShareRule
	require.NoError(t, json.NewDecoder(w.Body).Decode(&rules))
	assert.Len(t, rules, 1)

	// Delete rule
	req = httptest.NewRequest(http.MethodDelete, "/api/auto-share-rules/"+rule.ID, nil)
	req.SetPathValue("id", rule.ID)
	w = httptest.NewRecorder()
	srv.DeleteAutoShareRule(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAutoShareRule_MissingEmail(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	body := `{"recipient_email":"","expiry_hours":48}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+proj.ID+"/auto-share-rules", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("projectID", proj.ID)
	w := httptest.NewRecorder()
	srv.CreateAutoShareRule(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
