package handler_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
)

func TestCustomFieldCRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	// Create custom field definition
	body := `{"name":"Diagnosis","field_type":"text","required":false}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+proj.ID+"/custom-fields", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", proj.ID)
	w := httptest.NewRecorder()
	srv.CreateCustomFieldDefinition(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "Diagnosis")

	// List custom field definitions
	req = httptest.NewRequest(http.MethodGet, "/api/projects/"+proj.ID+"/custom-fields", nil)
	req.SetPathValue("id", proj.ID)
	w = httptest.NewRecorder()
	srv.ListCustomFieldDefinitions(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Diagnosis")
}

func TestCustomField_MissingName(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	body := `{"field_type":"text"}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+proj.ID+"/custom-fields", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", proj.ID)
	w := httptest.NewRecorder()
	srv.CreateCustomFieldDefinition(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCustomField_InvalidType(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	body := `{"name":"BadField","field_type":"boolean"}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+proj.ID+"/custom-fields", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", proj.ID)
	w := httptest.NewRecorder()
	srv.CreateCustomFieldDefinition(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
