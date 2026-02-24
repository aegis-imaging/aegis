package middleware

import (
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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// package-level test key — generated once, reused across all unit tests.
var (
	testPrivKey *ecdsa.PrivateKey
	testKid     = "test-kid-001"
)

func init() {
	var err error
	testPrivKey, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic("failed to generate test ECDSA key: " + err.Error())
	}
}

// makeSignedJWT builds a real ES256-signed JWT with the given payload claims.
// The header always sets alg=ES256 and kid=testKid.
func makeSignedJWT(t *testing.T, payload map[string]interface{}) string {
	t.Helper()

	headerJSON, _ := json.Marshal(map[string]string{"alg": "ES256", "kid": testKid, "typ": "JWT"})
	payloadJSON, _ := json.Marshal(payload)

	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)
	message := headerB64 + "." + payloadB64

	digest := sha256.Sum256([]byte(message))
	r, s, err := ecdsa.Sign(rand.Reader, testPrivKey, digest[:])
	require.NoError(t, err)

	// Encode as raw r||s (64 bytes) — the format ALB uses.
	sigBytes := make([]byte, 64)
	rBytes := r.Bytes()
	sBytes := s.Bytes()
	copy(sigBytes[32-len(rBytes):32], rBytes)
	copy(sigBytes[64-len(sBytes):64], sBytes)

	return headerB64 + "." + payloadB64 + "." + base64.RawURLEncoding.EncodeToString(sigBytes)
}

// startTestKeyServer spins up an httptest.Server serving testPrivKey's public
// key PEM at any path. It overrides ALBKeyBaseURL and clears albKeyCache for t.
func startTestKeyServer(t *testing.T) {
	t.Helper()

	pubDER, err := x509.MarshalPKIXPublicKey(&testPrivKey.PublicKey)
	require.NoError(t, err)
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-pem-file")
		w.Write(pubPEM)
	}))

	origURL := ALBKeyBaseURL
	ALBKeyBaseURL = srv.URL + "/%s/%s"
	albKeyCache.Delete(testKid)

	t.Cleanup(func() {
		srv.Close()
		ALBKeyBaseURL = origURL
		albKeyCache.Delete(testKid)
	})
}

// ── extractEmailFromALBJWT unit tests ─────────────────────────────────────────

func TestExtractEmailFromALBJWT_Valid(t *testing.T) {
	startTestKeyServer(t)
	token := makeSignedJWT(t, map[string]interface{}{"email": "User@Example.COM"})
	email, err := extractEmailFromALBJWT("us-east-1", token)
	require.NoError(t, err)
	assert.Equal(t, "user@example.com", email, "should be lowercased")
}

func TestExtractEmailFromALBJWT_Whitespace(t *testing.T) {
	startTestKeyServer(t)
	token := makeSignedJWT(t, map[string]interface{}{"email": "  test@example.com  "})
	email, err := extractEmailFromALBJWT("us-east-1", token)
	require.NoError(t, err)
	assert.Equal(t, "test@example.com", email, "should be trimmed")
}

func TestExtractEmailFromALBJWT_MalformedJWT(t *testing.T) {
	_, err := extractEmailFromALBJWT("us-east-1", "not-a-jwt")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "malformed JWT")
}

func TestExtractEmailFromALBJWT_TwoPartsOnly(t *testing.T) {
	_, err := extractEmailFromALBJWT("us-east-1", "header.payload")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "malformed JWT")
}

func TestExtractEmailFromALBJWT_InvalidBase64Payload(t *testing.T) {
	// Valid ES256 header, garbage payload — should fail at payload decode.
	headerJSON, _ := json.Marshal(map[string]string{"alg": "ES256", "kid": testKid})
	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	_, err := extractEmailFromALBJWT("us-east-1", headerB64+".!!!invalid!!!.sig")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode JWT payload")
}

func TestExtractEmailFromALBJWT_InvalidJSON(t *testing.T) {
	headerJSON, _ := json.Marshal(map[string]string{"alg": "ES256", "kid": testKid})
	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	payload := base64.RawURLEncoding.EncodeToString([]byte("not json"))
	_, err := extractEmailFromALBJWT("us-east-1", headerB64+"."+payload+".sig")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse JWT claims")
}

func TestExtractEmailFromALBJWT_NoEmailClaim(t *testing.T) {
	startTestKeyServer(t)
	token := makeSignedJWT(t, map[string]interface{}{"sub": "user123"})
	_, err := extractEmailFromALBJWT("us-east-1", token)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no email claim")
}

func TestExtractEmailFromALBJWT_EmptyEmailClaim(t *testing.T) {
	startTestKeyServer(t)
	token := makeSignedJWT(t, map[string]interface{}{"email": ""})
	_, err := extractEmailFromALBJWT("us-east-1", token)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no email claim")
}

func TestExtractEmailFromALBJWT_ExpiredJWT(t *testing.T) {
	startTestKeyServer(t)
	token := makeSignedJWT(t, map[string]interface{}{
		"email": "user@example.com",
		"exp":   int64(1000000000), // far in the past (2001-09-09)
	})
	_, err := extractEmailFromALBJWT("us-east-1", token)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "JWT has expired")
}

func TestExtractEmailFromALBJWT_InvalidSignature(t *testing.T) {
	startTestKeyServer(t)
	token := makeSignedJWT(t, map[string]interface{}{"email": "user@example.com"})
	parts := strings.SplitN(token, ".", 3)
	require.Len(t, parts, 3)

	// Replace payload with different content — signature is now over a different message.
	tamperedPayload := base64.RawURLEncoding.EncodeToString([]byte(`{"email":"attacker@evil.com"}`))
	tampered := parts[0] + "." + tamperedPayload + "." + parts[2]

	_, err := extractEmailFromALBJWT("us-east-1", tampered)
	require.Error(t, err)
	// Could fail at sig length check or at ecdsa.Verify — both are acceptable.
	assert.True(t,
		strings.Contains(err.Error(), "signature verification failed") ||
			strings.Contains(err.Error(), "unexpected signature length") ||
			strings.Contains(err.Error(), "decode JWT signature"),
		"unexpected error: %v", err)
}

func TestExtractEmailFromALBJWT_WrongAlgorithm(t *testing.T) {
	headerJSON, _ := json.Marshal(map[string]string{"alg": "RS256", "kid": testKid})
	payloadJSON, _ := json.Marshal(map[string]string{"email": "user@example.com"})
	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)
	token := headerB64 + "." + payloadB64 + ".fakesig"

	_, err := extractEmailFromALBJWT("us-east-1", token)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected JWT algorithm")
}

func TestExtractEmailFromALBJWT_MissingKid(t *testing.T) {
	headerJSON, _ := json.Marshal(map[string]string{"alg": "ES256"})
	payloadJSON, _ := json.Marshal(map[string]string{"email": "user@example.com"})
	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)
	token := headerB64 + "." + payloadB64 + ".fakesig"

	_, err := extractEmailFromALBJWT("us-east-1", token)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing kid")
}

// ── getALBPublicKey caching test ───────────────────────────────────────────────

func TestGetALBPublicKey_CachesKey(t *testing.T) {
	albKeyCache.Delete(testKid)

	fetchCount := 0
	pubDER, err := x509.MarshalPKIXPublicKey(&testPrivKey.PublicKey)
	require.NoError(t, err)
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fetchCount++
		w.Write(pubPEM)
	}))
	defer srv.Close()

	origURL := ALBKeyBaseURL
	ALBKeyBaseURL = srv.URL + "/%s/%s"
	t.Cleanup(func() {
		ALBKeyBaseURL = origURL
		albKeyCache.Delete(testKid)
	})

	k1, err := getALBPublicKey("us-east-1", testKid)
	require.NoError(t, err)
	assert.NotNil(t, k1)
	assert.Equal(t, 1, fetchCount, "should have fetched once")

	k2, err := getALBPublicKey("us-east-1", testKid)
	require.NoError(t, err)
	assert.Equal(t, k1, k2, "should return same cached pointer")
	assert.Equal(t, 1, fetchCount, "should not have fetched again")
}

// ── parseECPublicKey tests ─────────────────────────────────────────────────────

func TestParseECPublicKey_Valid(t *testing.T) {
	pubDER, err := x509.MarshalPKIXPublicKey(&testPrivKey.PublicKey)
	require.NoError(t, err)
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})

	key, err := parseECPublicKey(pubPEM)
	require.NoError(t, err)
	assert.NotNil(t, key)
}

func TestParseECPublicKey_InvalidPEM(t *testing.T) {
	_, err := parseECPublicKey([]byte("not pem"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "PEM")
}
