package middleware_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aegis-imaging/aegis/api/config"
	"github.com/aegis-imaging/aegis/api/middleware"
	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// integration test key — separate from the unit test key to keep tests independent.
var (
	intPrivKey *ecdsa.PrivateKey
	intKid     = "int-test-kid-001"
)

func init() {
	var err error
	intPrivKey, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic("failed to generate integration test ECDSA key: " + err.Error())
	}
}

// makeALBJWT builds a real ES256-signed JWT suitable for X-Amzn-Oidc-Data.
func makeALBJWT(t *testing.T, email string) string {
	t.Helper()

	headerJSON, _ := json.Marshal(map[string]string{"alg": "ES256", "kid": intKid, "typ": "JWT"})
	payloadJSON, _ := json.Marshal(map[string]string{"email": email})

	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)
	message := headerB64 + "." + payloadB64

	digest := sha256.Sum256([]byte(message))
	r, s, err := ecdsa.Sign(rand.Reader, intPrivKey, digest[:])
	require.NoError(t, err)

	sigBytes := make([]byte, 64)
	rBytes := r.Bytes()
	sBytes := s.Bytes()
	copy(sigBytes[32-len(rBytes):32], rBytes)
	copy(sigBytes[64-len(sBytes):64], sBytes)

	return headerB64 + "." + payloadB64 + "." + base64.RawURLEncoding.EncodeToString(sigBytes)
}

// startIntTestKeyServer overrides middleware.ALBKeyBaseURL with a local server
// serving intPrivKey's public key PEM. Clears the key cache for intKid.
func startIntTestKeyServer(t *testing.T) {
	t.Helper()

	pubDER, err := x509.MarshalPKIXPublicKey(&intPrivKey.PublicKey)
	require.NoError(t, err)
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-pem-file")
		w.Write(pubPEM)
	}))

	origURL := middleware.ALBKeyBaseURL
	middleware.ALBKeyBaseURL = srv.URL + "/%s/%s"

	t.Cleanup(func() {
		srv.Close()
		middleware.ALBKeyBaseURL = origURL
	})
}

func TestRequireAuth_AWSProvider(t *testing.T) {
	startIntTestKeyServer(t)

	db := testutil.TestDB(t)
	_ = model.CreateAdminUser(context.Background(), db, &model.AdminUser{
		Email: "aws-user@example.com", Name: "AWS User", Role: "admin", Enabled: true,
	})

	cfg := &config.Config{AuthEnabled: true, AuthProvider: "aws", AWSALBRegion: "us-east-1"}

	var capturedUser *middleware.AuthUser
	wrapped := middleware.RequireAuth(db, cfg)(func(w http.ResponseWriter, r *http.Request) {
		capturedUser = middleware.UserFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Amzn-Oidc-Data", makeALBJWT(t, "AWS-USER@Example.com"))
	rr := httptest.NewRecorder()
	wrapped(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	if assert.NotNil(t, capturedUser) {
		assert.Equal(t, "aws-user@example.com", capturedUser.Email)
		assert.Equal(t, "admin", capturedUser.Role)
	}
}

func TestRequireAuth_AWSProviderMissingHeader(t *testing.T) {
	db := testutil.TestDB(t)
	cfg := &config.Config{AuthEnabled: true, AuthProvider: "aws", AWSALBRegion: "us-east-1"}

	wrapped := middleware.RequireAuth(db, cfg)(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()
	wrapped(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Contains(t, rr.Body.String(), "missing AWS ALB authentication header")
}

func TestRequireAuth_AutoProviderWithAWSHeader(t *testing.T) {
	startIntTestKeyServer(t)

	db := testutil.TestDB(t)
	_ = model.CreateAdminUser(context.Background(), db, &model.AdminUser{
		Email: "auto-aws@example.com", Name: "Auto AWS", Role: "viewer", Enabled: true,
	})

	cfg := &config.Config{AuthEnabled: true, AuthProvider: "auto", AWSALBRegion: "us-east-1"}

	var capturedUser *middleware.AuthUser
	wrapped := middleware.RequireAuth(db, cfg)(func(w http.ResponseWriter, r *http.Request) {
		capturedUser = middleware.UserFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Amzn-Oidc-Data", makeALBJWT(t, "auto-aws@example.com"))
	rr := httptest.NewRecorder()
	wrapped(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	if assert.NotNil(t, capturedUser) {
		assert.Equal(t, "auto-aws@example.com", capturedUser.Email)
		assert.Equal(t, "viewer", capturedUser.Role)
	}
}

func TestRequireAuth_AWSProvider_TamperedJWT(t *testing.T) {
	startIntTestKeyServer(t)

	db := testutil.TestDB(t)
	cfg := &config.Config{AuthEnabled: true, AuthProvider: "aws", AWSALBRegion: "us-east-1"}

	wrapped := middleware.RequireAuth(db, cfg)(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Build a valid JWT, then tamper with the payload to inject a different email.
	validToken := makeALBJWT(t, "legitimate@example.com")
	parts := splitJWT(validToken)
	require.Len(t, parts, 3)

	tamperedPayload := base64.RawURLEncoding.EncodeToString([]byte(`{"email":"attacker@evil.com"}`))
	tampered := parts[0] + "." + tamperedPayload + "." + parts[2]

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Amzn-Oidc-Data", tampered)
	rr := httptest.NewRecorder()
	wrapped(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	body := rr.Body.String()
	assert.True(t,
		containsAny(body, "signature verification failed", "unexpected signature length", "decode JWT signature"),
		"expected signature error in: %s", body)
}

// splitJWT splits a JWT into its three parts without any validation.
func splitJWT(token string) []string {
	parts := make([]string, 0, 3)
	start := 0
	dots := 0
	for i, c := range token {
		if c == '.' {
			parts = append(parts, token[start:i])
			start = i + 1
			dots++
			if dots == 2 {
				parts = append(parts, token[start:])
				return parts
			}
		}
	}
	return parts
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if len(s) >= len(sub) {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}
