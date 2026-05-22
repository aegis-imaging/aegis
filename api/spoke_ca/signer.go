// Package spoke_ca signs CSRs from spoke routers so the cloud's mTLS upload
// path can authenticate them. Three deployment modes:
//
//  1. Configured PKI (prod): SPOKE_CA_CERT_PATH + SPOKE_CA_KEY_PATH point at a
//     real intermediate CA (sitting under your existing root) whose private
//     key the API can read. The right pick for an enterprise.
//
//  2. Auto-generated ephemeral CA (dev/test): both paths empty + AUTH_ENABLED
//     false. The signer mints a self-signed CA at process start, holds it in
//     memory, and signs from there. Useful for self-host installs and CI but
//     not durable across restarts.
//
//  3. Disabled (legacy compat): SPOKE_CA_ENABLED=false. /api/spokes/enroll
//     returns 503. Admins can still provision certs out-of-band and post the
//     thumbprint to PUT /api/institutions/{id}/client-cert.
package spoke_ca

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strings"
	"sync"
	"time"
)

// Signer wraps a CA cert + private key and signs spoke client CSRs.
type Signer struct {
	mu       sync.Mutex
	caCert   *x509.Certificate
	caKey    any // crypto.Signer (ecdsa.PrivateKey or rsa.PrivateKey)
	caPEM    []byte
	validity time.Duration
	source   string // "configured" | "ephemeral"
}

// SignedCert is the returned bundle from Sign.
type SignedCert struct {
	ClientCertPEM  string
	CAPEM          string
	Thumbprint     string // lowercase hex sha256 of the cert DER
	SubjectDN      string
	NotAfter       time.Time
	SerialHex      string
}

// Load builds a Signer from configured PEM paths. If both paths are empty,
// returns nil, nil — callers should then call NewEphemeral if they want to
// enable enrollment anyway.
func Load(certPath, keyPath string, validityDays int) (*Signer, error) {
	certPath = strings.TrimSpace(certPath)
	keyPath = strings.TrimSpace(keyPath)
	if certPath == "" && keyPath == "" {
		return nil, nil
	}
	if certPath == "" || keyPath == "" {
		return nil, errors.New("SPOKE_CA_CERT_PATH and SPOKE_CA_KEY_PATH must both be set, or both unset")
	}
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("read CA cert: %w", err)
	}
	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("read CA key: %w", err)
	}
	caCert, err := parseCertPEM(certPEM)
	if err != nil {
		return nil, fmt.Errorf("parse CA cert: %w", err)
	}
	caKey, err := parseKeyPEM(keyPEM)
	if err != nil {
		return nil, fmt.Errorf("parse CA key: %w", err)
	}
	return &Signer{
		caCert:   caCert,
		caKey:    caKey,
		caPEM:    certPEM,
		validity: validityDuration(validityDays),
		source:   "configured",
	}, nil
}

// NewEphemeral mints an in-memory self-signed CA. Cert + key never touch
// disk; logs only the fingerprint so an operator can verify it later.
func NewEphemeral(validityDays int) (*Signer, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	caTpl := &x509.Certificate{
		SerialNumber: bigSerial(),
		Subject: pkix.Name{
			CommonName:   "AEGIS Spoke Issuer (ephemeral)",
			Organization: []string{"AEGIS"},
		},
		NotBefore:             now.Add(-10 * time.Minute),
		NotAfter:              now.Add(10 * 365 * 24 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, caTpl, caTpl, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}
	caCert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, err
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	return &Signer{
		caCert:   caCert,
		caKey:    key,
		caPEM:    pemBytes,
		validity: validityDuration(validityDays),
		source:   "ephemeral",
	}, nil
}

// Source returns "configured" or "ephemeral" for diagnostics.
func (s *Signer) Source() string { return s.source }

// CAPEM returns the CA cert PEM bundle that should be returned to enrolled
// spokes so they can validate the cloud's serving cert (if the cloud is on
// the same chain).
func (s *Signer) CAPEM() string { return string(s.caPEM) }

// Sign validates the CSR's signature and issues a client cert with a Subject
// derived from (csr.CN OR fallbackCN) and the cert valid for the configured
// duration.
func (s *Signer) Sign(csrPEM string, fallbackCN string) (*SignedCert, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	block, _ := pem.Decode([]byte(csrPEM))
	if block == nil {
		return nil, errors.New("csr_pem is not a valid PEM block")
	}
	csr, err := x509.ParseCertificateRequest(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse CSR: %w", err)
	}
	if err := csr.CheckSignature(); err != nil {
		return nil, fmt.Errorf("CSR signature invalid: %w", err)
	}

	cn := strings.TrimSpace(csr.Subject.CommonName)
	if cn == "" {
		cn = strings.TrimSpace(fallbackCN)
	}
	if cn == "" {
		return nil, errors.New("CSR has no CommonName and no fallback was provided")
	}

	now := time.Now().UTC()
	tpl := &x509.Certificate{
		SerialNumber: bigSerial(),
		Subject: pkix.Name{
			CommonName:   cn,
			Organization: []string{"AEGIS Spoke"},
		},
		NotBefore:   now.Add(-5 * time.Minute),
		NotAfter:    now.Add(s.validity),
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tpl, s.caCert, csr.PublicKey, s.caKey)
	if err != nil {
		return nil, fmt.Errorf("sign cert: %w", err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, err
	}
	thumb := sha256.Sum256(der)
	return &SignedCert{
		ClientCertPEM: string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})),
		CAPEM:         string(s.caPEM),
		Thumbprint:    hex.EncodeToString(thumb[:]),
		SubjectDN:     cert.Subject.String(),
		NotAfter:      cert.NotAfter,
		SerialHex:     cert.SerialNumber.Text(16),
	}, nil
}

// ── helpers ────────────────────────────────────────────────────────────────

func parseCertPEM(b []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(b)
	if block == nil {
		return nil, errors.New("no PEM block")
	}
	return x509.ParseCertificate(block.Bytes)
}

func parseKeyPEM(b []byte) (any, error) {
	block, _ := pem.Decode(b)
	if block == nil {
		return nil, errors.New("no PEM block")
	}
	if k, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		return k, nil
	}
	if k, err := x509.ParseECPrivateKey(block.Bytes); err == nil {
		return k, nil
	}
	if k, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return k, nil
	}
	return nil, errors.New("unrecognised private key encoding (need PKCS8, EC, or PKCS1)")
}

func bigSerial() *big.Int {
	// 128-bit random serial — RFC 5280 requires <=20 octets.
	max := new(big.Int).Lsh(big.NewInt(1), 128)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		// Extremely unlikely; fall back to a deterministic-ish value rather
		// than panicking — Sign will succeed and the operator can investigate.
		return big.NewInt(time.Now().UnixNano())
	}
	return n
}

func validityDuration(days int) time.Duration {
	if days <= 0 {
		days = 365
	}
	return time.Duration(days) * 24 * time.Hour
}
