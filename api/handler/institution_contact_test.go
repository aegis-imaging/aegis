package handler_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
)

func TestInstitutionContactCRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	inst := testutil.CreateTestInstitution(t, db, "contact-test-inst")

	// Create contact
	body := `{"name":"Dr. Smith","email":"smith@example.com","phone":"555-1234","role":"PI"}`
	req := httptest.NewRequest(http.MethodPost, "/api/institutions/"+inst.ID+"/contacts", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", inst.ID)
	w := httptest.NewRecorder()
	srv.CreateInstitutionContact(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	// List contacts
	req = httptest.NewRequest(http.MethodGet, "/api/institutions/"+inst.ID+"/contacts", nil)
	req.SetPathValue("id", inst.ID)
	w = httptest.NewRecorder()
	srv.ListInstitutionContacts(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Dr. Smith")
}

func TestInstitutionContact_MissingName(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	inst := testutil.CreateTestInstitution(t, db, "contact-test-inst2")

	body := `{"email":"smith@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/api/institutions/"+inst.ID+"/contacts", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", inst.ID)
	w := httptest.NewRecorder()
	srv.CreateInstitutionContact(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
