package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDestinationProbeScheduler_WritesAuditEntries verifies that the scheduler
// probes enabled destinations and writes destination.tested audit entries.
func TestDestinationProbeScheduler_WritesAuditEntries(t *testing.T) {
	if testing.Short() {
		t.Log("integration test")
		t.Skip()
	}

	// Spin up a fake DICOMweb endpoint that returns 200.
	fakeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer fakeServer.Close()

	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	dest := testutil.CreateTestDestination(t, db, "Probe Target")

	// Point the destination at the fake server.
	_, err := db.Exec(`UPDATE destinations SET dicomweb_url = $1 WHERE id = $2`, fakeServer.URL, dest.ID)
	require.NoError(t, err)

	// Run one probe cycle synchronously.
	ctx := context.Background()
	srv.StartDestinationProbeScheduler(ctx, 1*time.Hour, "") // 1-hour interval so it won't fire again in test
	// Give the immediate first-run goroutine a moment to complete.
	time.Sleep(200 * time.Millisecond)

	// Verify an audit entry was written.
	var count int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM audit_trail
		WHERE action = 'destination.tested'
		  AND resource_id = $1
		  AND actor = 'scheduler'
		  AND (detail->>'auto')::boolean = true`,
		dest.ID).Scan(&count)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, count, 1, "scheduler should write at least one audit entry")
}

// TestDestinationProbeScheduler_Disabled verifies no goroutine is started when interval=0.
func TestDestinationProbeScheduler_Disabled(t *testing.T) {
	if testing.Short() {
		t.Log("integration test")
		t.Skip()
	}

	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	testutil.CreateTestDestination(t, db, "No Probe Dest")

	ctx := context.Background()
	// interval=0 means disabled — call should return immediately with no goroutine.
	srv.StartDestinationProbeScheduler(ctx, 0, "")
	time.Sleep(100 * time.Millisecond)

	var count int
	_ = db.QueryRow(`SELECT COUNT(*) FROM audit_trail WHERE actor = 'scheduler'`).Scan(&count)
	assert.Equal(t, 0, count, "disabled scheduler should not write any audit entries")
}

// TestDestinationProbeScheduler_FailingEndpoint verifies a failing probe is recorded correctly.
func TestDestinationProbeScheduler_FailingEndpoint(t *testing.T) {
	if testing.Short() {
		t.Log("integration test")
		t.Skip()
	}

	// Fake server returns 503.
	fakeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer fakeServer.Close()

	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	dest := testutil.CreateTestDestination(t, db, "Failing Dest")

	_, err := db.Exec(`UPDATE destinations SET dicomweb_url = $1 WHERE id = $2`, fakeServer.URL, dest.ID)
	require.NoError(t, err)

	// Manually run a probe cycle (not via scheduler, to avoid timing issues).
	ctx := context.Background()
	srv.StartDestinationProbeScheduler(ctx, 1*time.Hour, "")
	time.Sleep(300 * time.Millisecond)

	// Verify audit entry records failure.
	var successRaw []byte
	err = db.QueryRow(`
		SELECT detail->'success' FROM audit_trail
		WHERE action = 'destination.tested' AND resource_id = $1 AND actor = 'scheduler'
		ORDER BY created_at DESC LIMIT 1`, dest.ID).Scan(&successRaw)
	require.NoError(t, err)
	assert.Equal(t, "false", string(successRaw))
}

// TestDestination_UsesProbeHelper verifies the manual TestDestination HTTP handler
// still works correctly after refactoring to use probeDestination.
func TestDestination_UsesProbeHelper(t *testing.T) {
	if testing.Short() {
		t.Log("integration test")
		t.Skip()
	}

	fakeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer fakeServer.Close()

	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	dest := testutil.CreateTestDestination(t, db, "Manual Test Dest")

	_, err := db.Exec(`UPDATE destinations SET dicomweb_url = $1 WHERE id = $2`, fakeServer.URL, dest.ID)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/destinations/"+dest.ID+"/test", nil)
	req.SetPathValue("id", dest.ID)
	w := httptest.NewRecorder()
	srv.TestDestination(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Success   bool    `json:"success"`
		LatencyMs float64 `json:"latency_ms"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp.Success)
	assert.GreaterOrEqual(t, resp.LatencyMs, 0.0)

	// Verify audit entry written by handler (actor = "" for dev mode auto-auth).
	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM audit_trail WHERE action = 'destination.tested' AND resource_id = $1`, dest.ID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}
