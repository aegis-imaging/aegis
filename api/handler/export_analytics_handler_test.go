package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetExportAnalytics_OK(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest("GET", "/api/export-analytics", nil)
	rr := httptest.NewRecorder()
	srv.GetExportAnalytics(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var analytics model.DownloadAnalytics
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&analytics))
	// Empty state: zero downloads
	assert.Equal(t, 0, analytics.TotalDownloads)
	assert.NotNil(t, analytics.Last30Days)
	assert.NotNil(t, analytics.TopShares)
}
