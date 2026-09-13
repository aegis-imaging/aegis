// Satellite mTLS middleware — identifies satellite-site routers by their client cert.
//
// In production, TLS is typically terminated at a reverse proxy (GCP HTTPS LB,
// AWS ALB, nginx, etc.). The proxy is expected to forward the validated peer
// certificate as one of:
//
//   X-Client-Cert            : URL-encoded PEM (nginx default with $ssl_client_escaped_cert)
//   X-Client-Cert-Fingerprint: hex SHA-256 fingerprint (some load balancers only forward this)
//
// We compute the SHA-256 thumbprint over the DER-encoded cert and look up the
// institution by it. On success we attach a SatelliteIdentity to the request
// context; downstream handlers (e.g. upload-init) can use it to attribute the
// incoming study to the right satellite without an API key.
//
// If no cert is present, the middleware is a no-op — the request continues
// down the normal auth chain (browser session, API key, IP allowlist).
package middleware

import (
	"context"
	"crypto/sha256"
	"crypto/x509"
	"database/sql"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/aegis-imaging/aegis/api/model"
)

const (
	headerClientCert        = "X-Client-Cert"
	headerClientCertFP      = "X-Client-Cert-Fingerprint"
	headerSatelliteSiteAdvisory = "X-Aegis-Satellite-Site"
)

// SatelliteIdentity is what the middleware attaches to ctx on success.
type SatelliteIdentity struct {
	Institution        *model.Institution
	CertThumbprint     string // lowercase hex sha256
	CertSubjectDN      string
	SiteAdvisory       string // value of X-Aegis-Satellite-Site (informational)
	ReadDirectlyFromTLS bool
}

type satelliteContextKey struct{}

// SatelliteIdentityContextKey returns the context key used for SatelliteIdentity.
// Exposed for tests.
func SatelliteIdentityContextKey() any { return satelliteContextKey{} }

// SatelliteFromContext returns the SatelliteIdentity, if any.
func SatelliteFromContext(ctx context.Context) *SatelliteIdentity {
	if v, ok := ctx.Value(satelliteContextKey{}).(*SatelliteIdentity); ok {
		return v
	}
	return nil
}

// WithSatelliteMTLS wraps the next handler with satellite-mTLS context attachment.
// The middleware never rejects; it just populates context when a valid cert
// is presented. Authorization decisions stay with the downstream handler so
// the same chain can accept browser uploads, API-key uploads, and spoke
// uploads from one endpoint.
func WithSatelliteMTLS(db *sql.DB) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ident := identifySatellite(r, db)
			if ident != nil {
				r = r.WithContext(context.WithValue(r.Context(), satelliteContextKey{}, ident))
			}
			next(w, r)
		}
	}
}

func identifySatellite(r *http.Request, db *sql.DB) *SatelliteIdentity {
	thumb, subj, fromTLS, ok := extractCertThumbprint(r)
	if !ok {
		return nil
	}
	inst, err := model.GetInstitutionByClientCertThumbprint(r.Context(), db, thumb)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			// Don't fail the request; just log via context-less return.
			// (Caller-side audit handler would log if it cares.)
		}
		return nil
	}
	return &SatelliteIdentity{
		Institution:         inst,
		CertThumbprint:      thumb,
		CertSubjectDN:       subj,
		SiteAdvisory:        strings.TrimSpace(r.Header.Get(headerSatelliteSiteAdvisory)),
		ReadDirectlyFromTLS: fromTLS,
	}
}

// extractCertThumbprint pulls a cert thumbprint from (in order):
//  1. r.TLS.PeerCertificates[0] (direct TLS termination, mostly dev)
//  2. X-Client-Cert PEM header (nginx, most reverse proxies)
//  3. X-Client-Cert-Fingerprint hex header (load balancers that don't forward
//     the full cert)
//
// Returns (thumbprint, subjectDN, fromTLS, ok).
func extractCertThumbprint(r *http.Request) (string, string, bool, bool) {
	if r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
		cert := r.TLS.PeerCertificates[0]
		return sha256Hex(cert.Raw), cert.Subject.String(), true, true
	}
	if pemHeader := r.Header.Get(headerClientCert); pemHeader != "" {
		cert := parseClientCertHeader(pemHeader)
		if cert != nil {
			return sha256Hex(cert.Raw), cert.Subject.String(), false, true
		}
	}
	if fp := strings.TrimSpace(r.Header.Get(headerClientCertFP)); fp != "" {
		return normalizeFingerprint(fp), "", false, true
	}
	return "", "", false, false
}

// parseClientCertHeader handles the URL-encoded PEM that nginx sends.
func parseClientCertHeader(raw string) *x509.Certificate {
	if decoded, err := url.QueryUnescape(raw); err == nil {
		raw = decoded
	}
	// Some proxies send the PEM with literal \t/\n; normalize.
	raw = strings.ReplaceAll(raw, "\t", "")
	if !strings.Contains(raw, "BEGIN CERTIFICATE") {
		return nil
	}
	block, _ := pem.Decode([]byte(raw))
	if block == nil {
		return nil
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil
	}
	return cert
}

func sha256Hex(der []byte) string {
	sum := sha256.Sum256(der)
	return hex.EncodeToString(sum[:])
}

func normalizeFingerprint(fp string) string {
	fp = strings.ToLower(strings.TrimSpace(fp))
	fp = strings.ReplaceAll(fp, ":", "")
	fp = strings.ReplaceAll(fp, " ", "")
	return fp
}
