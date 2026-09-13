package handler_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
)

func TestExportShareTemplateCRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	// Create template
	body := `{"name":"Standard 72h share","recipient_email":"lab@example.com","expiry_hours":72}`
	req := httptest.NewRequest(http.MethodPost, "/api/export-share-templates", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.CreateExportShareTemplate(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "Standard 72h share")

	// List templates
	req = httptest.NewRequest(http.MethodGet, "/api/export-share-templates", nil)
	w = httptest.NewRecorder()
	srv.ListExportShareTemplates(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Standard 72h share")
}

func TestExportShareTemplate_MissingName(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body := `{"recipient_email":"lab@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/api/export-share-templates", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.CreateExportShareTemplate(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
