package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
)

// flusher wraps httptest.ResponseRecorder and implements http.Flusher
// so the SSE handler can call Flush().
type flusher struct{ *httptest.ResponseRecorder }

func (f *flusher) Flush() {}

func TestStudyEvents_ContentTypeAndConnectedComment(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	rr := &flusher{httptest.NewRecorder()}

	// Use a cancellable context so the SSE handler returns promptly.
	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodGet, "/api/studies/events", nil).WithContext(ctx)

	done := make(chan struct{})
	go func() {
		defer close(done)
		srv.StudyEvents(rr, req)
	}()

	// Give the handler time to write the initial comment and set headers.
	time.Sleep(80 * time.Millisecond)

	assert.Equal(t, "text/event-stream", rr.Header().Get("Content-Type"))
	assert.Contains(t, rr.Body.String(), ": connected")

	cancel() // disconnect the client
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("SSE handler did not return after context cancel")
	}
}

func TestStudyEvents_ProjectFilter(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	rr := &flusher{httptest.NewRecorder()}

	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodGet, "/api/studies/events?project_id=proj-1", nil).WithContext(ctx)

	done := make(chan struct{})
	go func() {
		defer close(done)
		srv.StudyEvents(rr, req)
	}()

	time.Sleep(60 * time.Millisecond)

	// The connected comment must still appear regardless of filter.
	assert.True(t, strings.Contains(rr.Body.String(), ": connected"))

	cancel()
	<-done
}
