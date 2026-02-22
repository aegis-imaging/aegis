package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContactForm_Success(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db) // mailer is no-op when SMTP_HOST unset

	body := map[string]any{
		"name":    "Dr. Smith",
		"email":   "smith@hospital.org",
		"message": "We'd like to learn more about AEGIS for our radiology department.",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/contact", bytes.NewReader(b))
	rr := httptest.NewRecorder()

	srv.ContactForm(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var result map[string]bool
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.True(t, result["sent"])
}

func TestContactForm_WithAllFields(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body := map[string]any{
		"name":         "Dr. Jane Doe",
		"email":        "jane@medcenter.org",
		"organization": "Metro Medical Center",
		"role":         "Chief Radiologist",
		"message":      "Interested in a pilot program for brain MRI studies.",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/contact", bytes.NewReader(b))
	rr := httptest.NewRecorder()

	srv.ContactForm(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestContactForm_MissingName(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body := map[string]any{
		"email":   "test@example.com",
		"message": "Hello",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/contact", bytes.NewReader(b))
	rr := httptest.NewRecorder()

	srv.ContactForm(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "name is required")
}

func TestContactForm_MissingEmail(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body := map[string]any{
		"name":    "Test",
		"message": "Hello",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/contact", bytes.NewReader(b))
	rr := httptest.NewRecorder()

	srv.ContactForm(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "email is required")
}

func TestContactForm_MissingMessage(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body := map[string]any{
		"name":  "Test",
		"email": "test@example.com",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/contact", bytes.NewReader(b))
	rr := httptest.NewRecorder()

	srv.ContactForm(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "message is required")
}

func TestContactForm_MessageTooLong(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body := map[string]any{
		"name":    "Test",
		"email":   "test@example.com",
		"message": strings.Repeat("x", 5001),
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/contact", bytes.NewReader(b))
	rr := httptest.NewRecorder()

	srv.ContactForm(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "message too long")
}

func TestContactForm_ExactMaxMessageLength(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body := map[string]any{
		"name":    "Test",
		"email":   "test@example.com",
		"message": strings.Repeat("x", 5000), // exactly at limit
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/contact", bytes.NewReader(b))
	rr := httptest.NewRecorder()

	srv.ContactForm(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestContactForm_InvalidJSON(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodPost, "/api/contact", bytes.NewReader([]byte(`{not json}`)))
	rr := httptest.NewRecorder()

	srv.ContactForm(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}
