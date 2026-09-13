package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// TestImportTCIASeries_AllowlistDenyShortCircuits verifies the
// allowlist short-circuit: when any institution linked to the
// destination project has browser.tcia-import disabled, the import
// returns 403 BEFORE attempting the TCIA download. This is the
// per-handler half of upload-allowlist chunk 2.
func TestImportTCIASeries_AllowlistDenyShortCircuits(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	// Build an institution → link to the default project → deny TCIA imports.
	inst := testutil.CreateTestInstitution(t, db, "tcia-deny")
	defaultProject, err := model.GetProjectBySlug(context.Background(), db, "default")
	require.NoError(t, err)
	require.NoError(t, model.AddInstitutionToProject(context.Background(), db,
		&model.InstitutionProject{InstitutionID: inst.ID, ProjectID: defaultProject.ID, Role: "sender"}))
	require.NoError(t, model.UpsertUploadAllowlistRow(context.Background(), db,
		&model.InstitutionUploadAllowlistRow{
			InstitutionID: inst.ID,
			MethodID:      "browser.tcia-import",
			Enabled:       false,
			Note:          "denied for the test",
			UpdatedBy:     "test@example.com",
		}))

	body, _ := json.Marshal(map[string]any{
		"series_uid":   fmt.Sprintf("1.2.3.tcia.deny.%d", time.Now().UnixNano()),
		"project_slug": "default",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/tcia/import", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.ImportTCIASeries(rr, req)

	// Should be 403, NOT 502 — proves we short-circuited before the TCIA
	// download attempt.
	assert.Equal(t, http.StatusForbidden, rr.Code, "body: %s", rr.Body.String())
	assert.Contains(t, rr.Body.String(), "disabled for your institution")
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
