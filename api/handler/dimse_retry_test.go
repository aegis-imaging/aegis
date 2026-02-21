package handler_test

import (
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/aegis-imaging/aegis/api/config"
	"github.com/aegis-imaging/aegis/api/handler"
	"github.com/aegis-imaging/aegis/api/storage"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func requireLoopbackListener(t *testing.T) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("skipping test: loopback listener unavailable: %v", err)
		return
	}
	_ = ln.Close()
}

func dimseProxyServer(t *testing.T, dimseURL, operatorKey string) *handler.Server {
	t.Helper()
	db := testutil.TestDB(t)
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Port:                "0",
		StorageMode:         "local",
		LocalStorageDir:     tmpDir,
		APIBaseURL:          "http://localhost:8080",
		PipelineAuto:        false,
		AuthEnabled:         false,
		DevUserEmail:        "test@aegis.local",
		AllowedOrigins:      []string{"http://localhost:3000"},
		DimseReceiverURL:    dimseURL,
		DimseOperatorAPIKey: operatorKey,
	}

	store := storage.NewLocal(tmpDir, cfg.APIBaseURL)
	return handler.NewServer(db, store, cfg)
}

func TestDimseRetryProxy_ServiceNotConfigured(t *testing.T) {
	srv := dimseProxyServer(t, "", "")

	req := httptest.NewRequest(http.MethodGet, "/api/dimse/retry/summary", nil)
	req.SetPathValue("path", "summary")
	rr := httptest.NewRecorder()
	srv.DimseRetryProxy(rr, req)

	assert.Equal(t, http.StatusServiceUnavailable, rr.Code)
	assert.Contains(t, rr.Body.String(), "dimse receiver service not configured")
}

func TestDimseRetryProxy_ProxiesGetWithQueryAndOperatorKey(t *testing.T) {
	requireLoopbackListener(t)

	var gotPath string
	var gotMethod string
	var gotOperatorKey string
	var gotLimit string
	var gotSort string
	var gotStudy string

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotOperatorKey = r.Header.Get("X-AEGIS-Operator-Key")
		gotLimit = r.URL.Query().Get("limit")
		gotSort = r.URL.Query().Get("sort")
		gotStudy = r.URL.Query().Get("study_instance_uid")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","ingest_retry":{"pending":1}}`))
	}))
	defer upstream.Close()

	srv := dimseProxyServer(t, upstream.URL, "op-key-1")
	req := httptest.NewRequest(
		http.MethodGet,
		"/api/dimse/retry/details?limit=25&sort=age_desc&study_instance_uid=1.2.3.4",
		nil,
	)
	req.SetPathValue("path", "details")
	rr := httptest.NewRecorder()
	srv.DimseRetryProxy(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.JSONEq(t, `{"status":"ok","ingest_retry":{"pending":1}}`, rr.Body.String())
	assert.Equal(t, http.MethodGet, gotMethod)
	assert.Equal(t, "/ingest/retry/details", gotPath)
	assert.Equal(t, "op-key-1", gotOperatorKey)
	assert.Equal(t, "25", gotLimit)
	assert.Equal(t, "age_desc", gotSort)
	assert.Equal(t, "1.2.3.4", gotStudy)
}

func TestDimseRetryProxy_ProxiesPostTargetedPath(t *testing.T) {
	requireLoopbackListener(t)

	var gotPath string
	var gotMethod string
	var gotQuery url.Values

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"status":"ok","ingest_retry":{"result":"requeued"}}`))
	}))
	defer upstream.Close()

	srv := dimseProxyServer(t, upstream.URL, "")
	req := httptest.NewRequest(http.MethodPost, "/api/dimse/retry/process/1.2.3.4?source=ui", strings.NewReader(""))
	req.SetPathValue("path", "process/1.2.3.4")
	rr := httptest.NewRecorder()
	srv.DimseRetryProxy(rr, req)

	assert.Equal(t, http.StatusAccepted, rr.Code)
	assert.JSONEq(t, `{"status":"ok","ingest_retry":{"result":"requeued"}}`, rr.Body.String())
	assert.Equal(t, http.MethodPost, gotMethod)
	assert.Equal(t, "/ingest/retry/process/1.2.3.4", gotPath)
	require.NotNil(t, gotQuery)
	assert.Equal(t, "ui", gotQuery.Get("source"))
}

func TestDimseRetryProxy_UpstreamUnavailable(t *testing.T) {
	srv := dimseProxyServer(t, "http://127.0.0.1:1", "")
	req := httptest.NewRequest(http.MethodGet, "/api/dimse/retry", nil)
	rr := httptest.NewRecorder()
	srv.DimseRetryProxy(rr, req)

	assert.Equal(t, http.StatusBadGateway, rr.Code)
	assert.Contains(t, rr.Body.String(), "failed to reach dimse receiver")
}
