package handler_test

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// readSSELines opens an SSE stream at tsURL and returns the first few lines
// that arrive before the context is cancelled.  Using a real HTTP round-trip
// (httptest.NewServer + http.Client) avoids the data races that occur when the
// handler goroutine and the test goroutine share an httptest.ResponseRecorder
// at the same time.
func readSSELines(t *testing.T, ctx context.Context, tsURL string) []string {
	t.Helper()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, tsURL, nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// context cancelled before the response arrived — that's fine
		return nil
	}
	defer resp.Body.Close()

	var lines []string
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		if len(lines) >= 5 {
			break // enough data; don't wait for more
		}
	}
	return lines
}

func TestStudyEvents_ContentTypeAndConnectedComment(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	ts := httptest.NewServer(http.HandlerFunc(srv.StudyEvents))
	defer ts.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	lines := readSSELines(t, ctx, ts.URL)

	// Verify Content-Type via a separate HEAD-like request so we can inspect headers.
	resp, err := http.Get(ts.URL) //nolint:noctx
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, "text/event-stream", resp.Header.Get("Content-Type"))

	// The very first line written is the ": connected" keepalive comment.
	assert.True(t, len(lines) > 0 && strings.Contains(strings.Join(lines, "\n"), ": connected"),
		"expected ': connected' in first SSE lines, got: %v", lines)
}

func TestStudyEvents_ProjectFilter(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	ts := httptest.NewServer(http.HandlerFunc(srv.StudyEvents))
	defer ts.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	lines := readSSELines(t, ctx, ts.URL+"?project_id=proj-1")

	// The connected comment must appear regardless of the project filter.
	all := strings.Join(lines, "\n")
	assert.Contains(t, all, ": connected")
}
