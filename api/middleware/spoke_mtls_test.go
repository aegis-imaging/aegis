package middleware_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/aegis-imaging/aegis/api/middleware"
	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeTestCert(t *testing.T, cn string) (pemBytes []byte, thumb string, subjectDN string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	tpl := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: cn, Organization: []string{"AEGIS Spoke"}},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, &key.PublicKey, key)
	require.NoError(t, err)
	pemBytes = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	sum := sha256.Sum256(der)
	thumb = hex.EncodeToString(sum[:])
	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	subjectDN = cert.Subject.String()
	return
}

func TestSpokeMTLS_NoCertHeaderPassesThrough(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	called := false
	h := middleware.WithSpokeMTLS(db)(func(w http.ResponseWriter, r *http.Request) {
		called = true
		assert.Nil(t, middleware.SpokeFromContext(r.Context()), "no spoke identity when no cert")
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodPost, "/api/upload/init", nil)
	rr := httptest.NewRecorder()
	h(rr, req)
	assert.True(t, called)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestSpokeMTLS_UnknownCertPassesThroughWithoutIdentity(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	pemBytes, _, _ := makeTestCert(t, "site-stranger")
	called := false
	h := middleware.WithSpokeMTLS(db)(func(w http.ResponseWriter, r *http.Request) {
		called = true
		assert.Nil(t, middleware.SpokeFromContext(r.Context()))
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodPost, "/api/upload/init", nil)
	req.Header.Set("X-Client-Cert", url.QueryEscape(string(pemBytes)))
	rr := httptest.NewRecorder()
	h(rr, req)
	assert.True(t, called)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestSpokeMTLS_KnownCertAttachesIdentity(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	inst := testutil.CreateTestInstitution(t, db, "mtls-known")

	pemBytes, thumb, subj := makeTestCert(t, "site-known")
	require.NoError(t, model.SetInstitutionClientCert(context.Background(), db, inst.ID, thumb, subj))

	var captured *middleware.SpokeIdentity
	h := middleware.WithSpokeMTLS(db)(func(w http.ResponseWriter, r *http.Request) {
		captured = middleware.SpokeFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodPost, "/api/upload/init", nil)
	req.Header.Set("X-Client-Cert", url.QueryEscape(string(pemBytes)))
	req.Header.Set("X-Aegis-Spoke-Site", "site-known")
	rr := httptest.NewRecorder()
	h(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	require.NotNil(t, captured, "expected spoke identity to be attached")
	assert.Equal(t, inst.ID, captured.Institution.ID)
	assert.Equal(t, thumb, captured.CertThumbprint)
	assert.Contains(t, captured.CertSubjectDN, "site-known")
	assert.Equal(t, "site-known", captured.SiteAdvisory)
}

func TestSpokeMTLS_FingerprintOnlyHeader(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	inst := testutil.CreateTestInstitution(t, db, "mtls-fp")

	_, thumb, subj := makeTestCert(t, "site-fp")
	require.NoError(t, model.SetInstitutionClientCert(context.Background(), db, inst.ID, thumb, subj))

	var captured *middleware.SpokeIdentity
	h := middleware.WithSpokeMTLS(db)(func(w http.ResponseWriter, r *http.Request) {
		captured = middleware.SpokeFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodPost, "/api/upload/init", nil)
	// Some proxies send fingerprint with colons.
	colons := strings.Join(chunked(thumb, 2), ":")
	req.Header.Set("X-Client-Cert-Fingerprint", colons)
	rr := httptest.NewRecorder()
	h(rr, req)
	require.NotNil(t, captured, "expected spoke identity from fingerprint header")
	assert.Equal(t, inst.ID, captured.Institution.ID)
	assert.Equal(t, thumb, captured.CertThumbprint)
}

func TestSpokeMTLS_DisabledInstitutionRejected(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	inst := testutil.CreateTestInstitution(t, db, "mtls-disabled")
	// Disable the institution.
	require.NoError(t, model.UpdateInstitution(context.Background(), db, &model.Institution{
		ID: inst.ID, Name: inst.Name, Slug: inst.Slug, Type: inst.Type, Enabled: false,
	}))
	pemBytes, thumb, subj := makeTestCert(t, "site-disabled")
	require.NoError(t, model.SetInstitutionClientCert(context.Background(), db, inst.ID, thumb, subj))

	var captured *middleware.SpokeIdentity
	h := middleware.WithSpokeMTLS(db)(func(w http.ResponseWriter, r *http.Request) {
		captured = middleware.SpokeFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodPost, "/api/upload/init", nil)
	req.Header.Set("X-Client-Cert", url.QueryEscape(string(pemBytes)))
	rr := httptest.NewRecorder()
	h(rr, req)
	assert.Nil(t, captured, "disabled institution should not produce a spoke identity")
}

func chunked(s string, n int) []string {
	out := make([]string, 0, len(s)/n+1)
	for i := 0; i < len(s); i += n {
		end := i + n
		if end > len(s) {
			end = len(s)
		}
		out = append(out, s[i:end])
	}
	return out
}
