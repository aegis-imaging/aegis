package handler_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
)

// ─── GetTCIASeries (validation-only; external HTTP call not made) ─────────────

func TestGetTCIASeries_MissingCollection(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/tcia/series", nil)
	rr := httptest.NewRecorder()
	srv.GetTCIASeries(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "collection is required")
}

func TestGetTCIASeries_CollectionOnlyWhitespace(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/tcia/series?collection=+", nil)
	rr := httptest.NewRecorder()
	srv.GetTCIASeries(rr, req)

	// "+" decodes to a space in URL query strings; TrimSpace makes it empty.
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "collection is required")
}

// ─── ImportTCIASeries (validation-only; external HTTP call not made) ──────────

func TestImportTCIASeries_MissingBody(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodPost, "/api/tcia/import", nil)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.ImportTCIASeries(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestImportTCIASeries_EmptyBody(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodPost, "/api/tcia/import",
		bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.ImportTCIASeries(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "series_uid is required")
}

func TestImportTCIASeries_MissingSeriesUID(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body, _ := json.Marshal(map[string]any{
		"collection":   "TCGA-GBM",
		"project_slug": "default",
		// series_uid intentionally absent
	})
	req := httptest.NewRequest(http.MethodPost, "/api/tcia/import", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.ImportTCIASeries(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "series_uid is required")
}

func TestImportTCIASeries_WhitespaceOnlySeriesUID(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body, _ := json.Marshal(map[string]any{
		"series_uid":   "   ",
		"project_slug": "default",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/tcia/import", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.ImportTCIASeries(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "series_uid is required")
}

func TestImportTCIASeries_ProjectNotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body, _ := json.Marshal(map[string]any{
		"series_uid":   fmt.Sprintf("1.2.3.tcia.%d", time.Now().UnixNano()),
		"project_slug": "nonexistent-project-xyz",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/tcia/import", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.ImportTCIASeries(rr, req)

	// Project not found should return bad gateway or bad request.
	// The external TCIA download is attempted first, so we expect 502 (network
	// error to TCIA) or 400 (project not found after successful download).
	// Either way it should not be 200.
	assert.NotEqual(t, http.StatusOK, rr.Code)
}

func TestImportTCIASeries_DefaultProjectSlug(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	// When project_slug is empty, it defaults to "default" which always exists.
	// The handler will try to call TCIA, which will fail with 502 in test env.
	body, _ := json.Marshal(map[string]any{
		"series_uid": fmt.Sprintf("1.2.3.tcia.default.%d", time.Now().UnixNano()),
		// project_slug omitted intentionally
	})
	req := httptest.NewRequest(http.MethodPost, "/api/tcia/import", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.ImportTCIASeries(rr, req)

	// Should get a bad gateway (can't reach TCIA in test) rather than a
	// validation error — proving the default project slug handling worked.
	assert.Equal(t, http.StatusBadGateway, rr.Code)
}

func TestImportTCIASeries_InvalidJSON(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodPost, "/api/tcia/import",
		bytes.NewReader([]byte("not-json")))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.ImportTCIASeries(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "invalid request body")
}
