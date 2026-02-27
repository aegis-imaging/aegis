package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostTestSignsPayloadWhenSecretPresent(t *testing.T) {
	secret := "top-secret"
	payload := Payload{
		Event:            "study.approved",
		StudyID:          "study-1",
		StudyInstanceUID: "1.2.3",
		ProjectID:        "project-1",
		Timestamp:        "2026-01-01T00:00:00Z",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "webhook", r.Header.Get("X-Aegis-Event"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(body)
		expectedSig := "sha256=" + hex.EncodeToString(mac.Sum(nil))
		assert.Equal(t, expectedSig, r.Header.Get("X-Aegis-Signature"))

		var got Payload
		err = json.Unmarshal(body, &got)
		require.NoError(t, err)
		assert.Equal(t, payload.Event, got.Event)
		assert.Equal(t, payload.StudyID, got.StudyID)

		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	status, err := PostTest(model.WebhookSubscription{
		URL:    server.URL,
		Secret: secret,
	}, payload)

	require.NoError(t, err)
	assert.Equal(t, http.StatusAccepted, status)
}

func TestPostTestReturnsErrorForNon2xx(t *testing.T) {
	payload := Payload{Event: "study.approved"}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	status, err := PostTest(model.WebhookSubscription{URL: server.URL}, payload)

	assert.Equal(t, http.StatusInternalServerError, status)
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "non-2xx response"))
}