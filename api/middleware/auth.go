package middleware

import (
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/aegis-imaging/aegis/api/config"
	"github.com/aegis-imaging/aegis/api/model"
)

// ALBKeyBaseURL is the format string used to fetch ALB public keys.
// fmt.Sprintf(ALBKeyBaseURL, region, kid) must produce a valid URL.
// Override in tests to point at a local httptest.Server.
var ALBKeyBaseURL = "https://public-keys.auth.elb.%s.amazonaws.com/%s"

var (
	albKeyCache   sync.Map // kid → *ecdsa.PublicKey
	albHTTPClient = &http.Client{Timeout: 5 * time.Second}
)

// AuthUser represents the authenticated user stored in the request context.
type AuthUser struct {
	ID    string
	Email string
	Name  string
	Role  string // "admin" or "viewer"
}

type contextKey string

const authUserKey contextKey = "auth_user"

// UserFromContext retrieves the authenticated user from the request context.
// Returns nil if no user is set (public routes).
func UserFromContext(ctx context.Context) *AuthUser {
	u, _ := ctx.Value(authUserKey).(*AuthUser)
	return u
}

// sessionDedupWindow is the minimum gap between recorded login sessions for the
// same user. Requests within this window do not create a new row.
const sessionDedupWindow = 30 * time.Minute

// RequireAuth returns a middleware that enforces authentication on a handler.
// When cfg.AuthEnabled is false (local dev), it auto-authenticates using
// cfg.DevUserEmail. When true (production), it extracts user identity from
// the configured auth provider's headers (GCP IAP or Azure AD Easy Auth).
func RequireAuth(db *sql.DB, cfg *config.Config) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			var user *AuthUser
			var err error

			if !cfg.AuthEnabled {
				user, err = devUser(r.Context(), db, cfg.DevUserEmail)
			} else {
				user, err = extractUser(r.Context(), db, r, cfg)
			}

			if err != nil {
				writeAuthError(w, err)
				return
			}

			// Record login session asynchronously, deduped per 30 minutes.
			// Only record for real users (non-synthetic dev user ID).
			if user.ID != "00000000-0000-0000-0000-000000000000" {
				ip := clientIPFromRequest(r)
				ua := r.UserAgent()
				uid := user.ID
				go func() {
					_ = model.RecordAdminSessionWithDedup(context.Background(), db, uid, ip, ua, sessionDedupWindow)
				}()
			}

			ctx := context.WithValue(r.Context(), authUserKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		}
	}
}

// clientIPFromRequest extracts the real client IP from the request, respecting
// X-Forwarded-For and X-Real-IP headers set by reverse proxies.
func clientIPFromRequest(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	host := r.RemoteAddr
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		return host[:idx]
	}
	return host
}

// RequireRole wraps RequireAuth and additionally checks that the authenticated
// user has the specified role. "admin" role has access to everything.
func RequireRole(role string, db *sql.DB, cfg *config.Config) func(http.HandlerFunc) http.HandlerFunc {
	authWrap := RequireAuth(db, cfg)
	return func(next http.HandlerFunc) http.HandlerFunc {
		return authWrap(func(w http.ResponseWriter, r *http.Request) {
			user := UserFromContext(r.Context())
			if user.Role != role && user.Role != "admin" {
				authWriteJSON(w, http.StatusForbidden, map[string]string{
					"error": "insufficient permissions: requires " + role + " role",
				})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

type authError struct {
	Status  int
	Message string
}

func (e *authError) Error() string { return e.Message }

// extractUser determines the authenticated user based on the configured provider.
// Supported providers: "iap" (GCP), "azure" (Azure AD), "aws" (ALB + Cognito), "auto" (try all).
func extractUser(ctx context.Context, db *sql.DB, r *http.Request, cfg *config.Config) (*AuthUser, error) {
	switch cfg.AuthProvider {
	case "iap":
		return iapUser(ctx, db, r)
	case "azure":
		return azureUser(ctx, db, r)
	case "aws":
		return awsUser(ctx, db, r, cfg.AWSALBRegion)
	default: // "auto" — try IAP, then Azure, then AWS
		if r.Header.Get("X-Goog-Authenticated-User-Email") != "" {
			return iapUser(ctx, db, r)
		}
		if r.Header.Get("X-MS-CLIENT-PRINCIPAL-NAME") != "" {
			return azureUser(ctx, db, r)
		}
		if r.Header.Get("X-Amzn-Oidc-Data") != "" {
			return awsUser(ctx, db, r, cfg.AWSALBRegion)
		}
		return nil, &authError{http.StatusUnauthorized, "missing authentication header"}
	}
}

func devUser(ctx context.Context, db *sql.DB, email string) (*AuthUser, error) {
	u, err := model.GetAdminUserByEmail(ctx, db, email)
	if err != nil {
		log.Printf("auth: dev user %q not found in admin_users, using synthetic admin", email)
		return &AuthUser{
			ID:    "00000000-0000-0000-0000-000000000000",
			Email: email,
			Name:  "Dev User",
			Role:  "admin",
		}, nil
	}
	if !u.Enabled {
		return nil, &authError{http.StatusForbidden, "account is disabled"}
	}
	return &AuthUser{ID: u.ID, Email: u.Email, Name: u.Name, Role: u.Role}, nil
}

// iapUser extracts user identity from GCP Identity-Aware Proxy headers.
// IAP sets X-Goog-Authenticated-User-Email to "accounts.google.com:user@example.com".
func iapUser(ctx context.Context, db *sql.DB, r *http.Request) (*AuthUser, error) {
	raw := r.Header.Get("X-Goog-Authenticated-User-Email")
	if raw == "" {
		return nil, &authError{http.StatusUnauthorized, "missing GCP IAP authentication header"}
	}

	// Strip "accounts.google.com:" prefix
	email := raw
	if idx := strings.Index(raw, ":"); idx != -1 {
		email = raw[idx+1:]
	}
	email = strings.TrimSpace(strings.ToLower(email))

	if email == "" {
		return nil, &authError{http.StatusUnauthorized, "invalid GCP IAP authentication header"}
	}

	return lookupUser(ctx, db, email)
}

// azureUser extracts user identity from Azure AD Easy Auth headers.
// Azure App Service Easy Auth sets X-MS-CLIENT-PRINCIPAL-NAME to the user's email.
// Azure Application Gateway + AAD also supports this header pattern.
func azureUser(ctx context.Context, db *sql.DB, r *http.Request) (*AuthUser, error) {
	email := r.Header.Get("X-MS-CLIENT-PRINCIPAL-NAME")
	if email == "" {
		return nil, &authError{http.StatusUnauthorized, "missing Azure AD authentication header"}
	}

	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return nil, &authError{http.StatusUnauthorized, "invalid Azure AD authentication header"}
	}

	return lookupUser(ctx, db, email)
}

// awsUser extracts user identity from AWS ALB + Cognito headers.
// AWS ALB sets X-Amzn-Oidc-Data (JWT) containing an "email" claim.
// The JWT is ES256-signed; the signature is verified against the ALB public key
// fetched from the regional public key endpoint and cached in memory.
func awsUser(ctx context.Context, db *sql.DB, r *http.Request, region string) (*AuthUser, error) {
	jwt := r.Header.Get("X-Amzn-Oidc-Data")
	if jwt == "" {
		return nil, &authError{http.StatusUnauthorized, "missing AWS ALB authentication header"}
	}

	email, err := extractEmailFromALBJWT(region, jwt)
	if err != nil {
		return nil, &authError{http.StatusUnauthorized, "invalid AWS ALB authentication token: " + err.Error()}
	}

	return lookupUser(ctx, db, email)
}

// extractEmailFromALBJWT pulls the "email" claim from the ALB-injected JWT payload
// and verifies the ES256 signature against the ALB regional public key endpoint.
// ALB uses the raw 64-byte r||s signature format (not ASN.1 DER).
func extractEmailFromALBJWT(region, token string) (string, error) {
	parts := strings.SplitN(token, ".", 4)
	if len(parts) < 3 {
		return "", errors.New("malformed JWT")
	}

	headerB64, payloadB64, sigB64 := parts[0], parts[1], parts[2]

	// ── 1. Decode and parse header ────────────────────────────────────────────
	headerBytes, err := base64.RawURLEncoding.DecodeString(headerB64)
	if err != nil {
		return "", fmt.Errorf("decode JWT header: %w", err)
	}
	var header struct {
		Alg string `json:"alg"`
		Kid string `json:"kid"`
	}
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return "", fmt.Errorf("parse JWT header: %w", err)
	}
	if header.Alg != "ES256" {
		return "", fmt.Errorf("unexpected JWT algorithm %q: expected ES256", header.Alg)
	}
	if header.Kid == "" {
		return "", errors.New("missing kid in JWT header")
	}

	// ── 2. Decode and parse payload ───────────────────────────────────────────
	// Pad to a multiple of 4 for standard Base64 decoding.
	paddedPayload := payloadB64
	switch len(paddedPayload) % 4 {
	case 2:
		paddedPayload += "=="
	case 3:
		paddedPayload += "="
	}
	payloadBytes, err := base64.URLEncoding.DecodeString(paddedPayload)
	if err != nil {
		return "", fmt.Errorf("decode JWT payload: %w", err)
	}
	var claims struct {
		Email string `json:"email"`
		Exp   int64  `json:"exp"`
	}
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return "", fmt.Errorf("parse JWT claims: %w", err)
	}

	// ── 3. Check expiry ───────────────────────────────────────────────────────
	if claims.Exp > 0 && time.Now().Unix() > claims.Exp {
		return "", errors.New("JWT has expired")
	}

	email := strings.TrimSpace(strings.ToLower(claims.Email))
	if email == "" {
		return "", errors.New("no email claim in JWT")
	}

	// ── 4. Fetch + cache public key ───────────────────────────────────────────
	pubKey, err := getALBPublicKey(region, header.Kid)
	if err != nil {
		return "", fmt.Errorf("fetch ALB public key: %w", err)
	}

	// ── 5. Verify signature ───────────────────────────────────────────────────
	// Strip any trailing padding characters before RawURLEncoding decode.
	// Some ALB JWT implementations include "==" padding in the signature section.
	sigBytes, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(sigB64, "="))
	if err != nil {
		return "", fmt.Errorf("decode JWT signature: %w", err)
	}
	if len(sigBytes) != 64 {
		return "", fmt.Errorf("unexpected signature length %d (expected 64)", len(sigBytes))
	}

	// ALB uses raw r||s (64 bytes) — NOT ASN.1 DER.
	r := new(big.Int).SetBytes(sigBytes[:32])
	s := new(big.Int).SetBytes(sigBytes[32:])

	// Message = header_b64 + "." + payload_b64 (the signed bytes).
	message := headerB64 + "." + payloadB64
	digest := sha256.Sum256([]byte(message))

	if !ecdsaVerify(pubKey, digest[:], r, s) {
		return "", errors.New("JWT signature verification failed")
	}

	return email, nil
}

// ecdsaVerify is a thin wrapper so tests can easily confirm we call the right function.
var ecdsaVerify = func(pub *ecdsa.PublicKey, hash []byte, r, s *big.Int) bool {
	return ecdsa.Verify(pub, hash, r, s)
}

// getALBPublicKey fetches and caches the ECDSA P-256 public key for the given kid.
func getALBPublicKey(region, kid string) (*ecdsa.PublicKey, error) {
	if cached, ok := albKeyCache.Load(kid); ok {
		return cached.(*ecdsa.PublicKey), nil
	}

	url := fmt.Sprintf(ALBKeyBaseURL, region, kid)
	resp, err := albHTTPClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: unexpected status %d", url, resp.StatusCode)
	}

	pemBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read key body: %w", err)
	}

	pub, err := parseECPublicKey(pemBytes)
	if err != nil {
		return nil, err
	}

	albKeyCache.Store(kid, pub)
	return pub, nil
}

// parseECPublicKey decodes a PEM-encoded ECDSA public key.
func parseECPublicKey(pemBytes []byte) (*ecdsa.PublicKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}
	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse PKIX public key: %w", err)
	}
	ecKey, ok := key.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("key is not ECDSA (got %T)", key)
	}
	return ecKey, nil
}

// lookupUser finds a user by email in admin_users and checks they are enabled.
func lookupUser(ctx context.Context, db *sql.DB, email string) (*AuthUser, error) {
	u, err := model.GetAdminUserByEmail(ctx, db, email)
	if err != nil {
		return nil, &authError{http.StatusForbidden, "user not registered: " + email}
	}
	if !u.Enabled {
		return nil, &authError{http.StatusForbidden, "account is disabled"}
	}
	return &AuthUser{ID: u.ID, Email: u.Email, Name: u.Name, Role: u.Role}, nil
}

func writeAuthError(w http.ResponseWriter, err error) {
	if ae, ok := err.(*authError); ok {
		authWriteJSON(w, ae.Status, map[string]string{"error": ae.Message})
		return
	}
	authWriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "authentication error"})
}

func authWriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
