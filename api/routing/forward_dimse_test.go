package routing

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/msenjem/aegis/api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestForwardStudyDIMSE_MissingURL(t *testing.T) {
	t.Setenv("DIMSE_RECEIVER_URL", "")
	study := &model.Study{StudyInstanceUID: "1.2.3.4", DicomStore: "raw"}
	dest := &model.Destination{Type: "dimse", AETitle: "REMOTE", Host: "127.0.0.1", Port: 104}

	err := forwardStudyDIMSE(context.Background(), study, dest)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DIMSE_RECEIVER_URL")
}

func TestForwardStudyDIMSE_Success(t *testing.T) {
	study := &model.Study{StudyInstanceUID: "1.2.3.4", DicomStore: "clean"}
	dest := &model.Destination{Type: "dimse", AETitle: "REMOTE_AE", Host: "10.0.0.8", Port: 11112}

	var got dimseForwardPayload
	oldClient := dimseForwardHTTPClient
	t.Cleanup(func() { dimseForwardHTTPClient = oldClient })
	dimseForwardHTTPClient = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/forward", r.URL.Path)
			require.NoError(t, json.NewDecoder(r.Body).Decode(&got))
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(`{"status":"complete"}`)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	t.Setenv("DIMSE_RECEIVER_URL", "http://dimse-receiver:8080")
	err := forwardStudyDIMSE(context.Background(), study, dest)
	require.NoError(t, err)

	assert.Equal(t, "1.2.3.4", got.StudyInstanceUID)
	assert.Equal(t, "clean", got.DicomStore)
	assert.Equal(t, "REMOTE_AE", got.Destination.AETitle)
	assert.Equal(t, "10.0.0.8", got.Destination.Host)
	assert.Equal(t, 11112, got.Destination.Port)
}

func TestForwardStudyDIMSE_HTTPError(t *testing.T) {
	study := &model.Study{StudyInstanceUID: "1.2.3.4", DicomStore: "raw"}
	dest := &model.Destination{Type: "dimse", AETitle: "REMOTE_AE", Host: "10.0.0.8", Port: 11112}

	oldClient := dimseForwardHTTPClient
	t.Cleanup(func() { dimseForwardHTTPClient = oldClient })
	dimseForwardHTTPClient = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusBadGateway,
				Body:       io.NopCloser(bytes.NewBufferString("association failed")),
				Header:     make(http.Header),
			}, nil
		}),
	}

	t.Setenv("DIMSE_RECEIVER_URL", "http://dimse-receiver:8080")
	err := forwardStudyDIMSE(context.Background(), study, dest)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 502")
}
