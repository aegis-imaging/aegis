package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/msenjem/aegis/api/config"
	"github.com/msenjem/aegis/api/handler"
	"github.com/stretchr/testify/assert"
)

func TestBatchImport_ReturnsBadRequestOnValidationError(t *testing.T) {
	// Use a definitely-invalid directory so importer.Run returns ValidationError
	// before touching DB or storage.
	body, _ := json.Marshal(map[string]any{
		"dir": "/definitely/missing/aegis/import/path",
	})

	req := httptest.NewRequest("POST", "/api/import/batch", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	srv := handler.NewServer(nil, nil, &config.Config{})
	srv.BatchImport(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestBatchImport_RejectsUnknownJSONFields(t *testing.T) {
	body, _ := json.Marshal(map[string]any{
		"dir":                  "/tmp",
		"institution_ae_title": "PACS_ALPHA",
	})

	req := httptest.NewRequest("POST", "/api/import/batch", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	srv := handler.NewServer(nil, nil, &config.Config{})
	srv.BatchImport(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}
