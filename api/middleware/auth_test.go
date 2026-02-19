package middleware

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeJWT(payload map[string]string) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"ES256"}`))
	payloadJSON, _ := json.Marshal(payload)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)
	return header + "." + payloadB64 + ".fakesig"
}

func TestExtractEmailFromALBJWT_Valid(t *testing.T) {
	token := makeJWT(map[string]string{"email": "User@Example.COM"})
	email, err := extractEmailFromALBJWT(token)
	require.NoError(t, err)
	assert.Equal(t, "user@example.com", email, "should be lowercased")
}

func TestExtractEmailFromALBJWT_Whitespace(t *testing.T) {
	token := makeJWT(map[string]string{"email": "  test@example.com  "})
	email, err := extractEmailFromALBJWT(token)
	require.NoError(t, err)
	assert.Equal(t, "test@example.com", email, "should be trimmed")
}

func TestExtractEmailFromALBJWT_MalformedJWT(t *testing.T) {
	_, err := extractEmailFromALBJWT("not-a-jwt")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "malformed JWT")
}

func TestExtractEmailFromALBJWT_TwoPartsOnly(t *testing.T) {
	_, err := extractEmailFromALBJWT("header.payload")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "malformed JWT")
}

func TestExtractEmailFromALBJWT_InvalidBase64(t *testing.T) {
	_, err := extractEmailFromALBJWT("header.!!!invalid!!!.sig")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode JWT payload")
}

func TestExtractEmailFromALBJWT_InvalidJSON(t *testing.T) {
	payload := base64.RawURLEncoding.EncodeToString([]byte("not json"))
	_, err := extractEmailFromALBJWT("header." + payload + ".sig")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse JWT claims")
}

func TestExtractEmailFromALBJWT_NoEmailClaim(t *testing.T) {
	token := makeJWT(map[string]string{"sub": "user123"})
	_, err := extractEmailFromALBJWT(token)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no email claim")
}

func TestExtractEmailFromALBJWT_EmptyEmailClaim(t *testing.T) {
	token := makeJWT(map[string]string{"email": ""})
	_, err := extractEmailFromALBJWT(token)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no email claim")
}
