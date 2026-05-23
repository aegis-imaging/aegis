package spoke_ca_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/aegis-imaging/aegis/api/spoke_ca"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeCSR(t *testing.T, cn string) string {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	tpl := &x509.CertificateRequest{
		Subject: pkix.Name{CommonName: cn, Organization: []string{"AEGIS Test"}},
	}
	der, err := x509.CreateCertificateRequest(rand.Reader, tpl, key)
	require.NoError(t, err)
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: der}))
}

func TestEphemeralSignerSignsCSR(t *testing.T) {
	signer, err := spoke_ca.NewEphemeral(30)
	require.NoError(t, err)
	assert.Equal(t, "ephemeral", signer.Source())

	csr := makeCSR(t, "site-alpha")
	signed, err := signer.Sign(csr, "")
	require.NoError(t, err)
	assert.Len(t, signed.Thumbprint, 64) // sha256 hex
	assert.Contains(t, signed.SubjectDN, "site-alpha")
	assert.Contains(t, signed.ClientCertPEM, "BEGIN CERTIFICATE")
	assert.Contains(t, signed.CAPEM, "BEGIN CERTIFICATE")
	assert.True(t, signed.NotAfter.After(signed.NotAfter.Add(-1)), "NotAfter set")
}

func TestSignFallsBackToProvidedCN(t *testing.T) {
	signer, err := spoke_ca.NewEphemeral(30)
	require.NoError(t, err)
	// CSR with empty CN.
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	tpl := &x509.CertificateRequest{}
	der, err := x509.CreateCertificateRequest(rand.Reader, tpl, key)
	require.NoError(t, err)
	csrPEM := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: der}))

	signed, err := signer.Sign(csrPEM, "fallback-site")
	require.NoError(t, err)
	assert.Contains(t, signed.SubjectDN, "fallback-site")
}

func TestSignRejectsInvalidPEM(t *testing.T) {
	signer, err := spoke_ca.NewEphemeral(30)
	require.NoError(t, err)
	_, err = signer.Sign("not a pem", "x")
	assert.Error(t, err)
}

func TestSignRejectsEmptyCN(t *testing.T) {
	signer, err := spoke_ca.NewEphemeral(30)
	require.NoError(t, err)
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	der, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{}, key)
	require.NoError(t, err)
	csrPEM := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: der}))

	_, err = signer.Sign(csrPEM, "")
	assert.Error(t, err)
}

func TestLoadNilWhenPathsEmpty(t *testing.T) {
	s, err := spoke_ca.Load("", "", 30)
	require.NoError(t, err)
	assert.Nil(t, s)
}

func TestLoadRequiresBothPaths(t *testing.T) {
	_, err := spoke_ca.Load("/tmp/cert", "", 30)
	assert.Error(t, err)
	_, err = spoke_ca.Load("", "/tmp/key", 30)
	assert.Error(t, err)
}

func TestLoadFromDisk(t *testing.T) {
	// Generate a CA on disk so we can verify Load handles real PEM files.
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	caTpl := &x509.Certificate{
		SerialNumber:          mustSerial(),
		Subject:               pkix.Name{CommonName: "test-ca"},
		NotBefore:             timeNow().Add(-1),
		NotAfter:              timeNow().AddDate(1, 0, 0),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
	}
	der, err := x509.CreateCertificate(rand.Reader, caTpl, caTpl, &key.PublicKey, key)
	require.NoError(t, err)
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)

	dir := t.TempDir()
	certPath := filepath.Join(dir, "ca.crt")
	keyPath := filepath.Join(dir, "ca.key")
	require.NoError(t, os.WriteFile(certPath,
		pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600))
	require.NoError(t, os.WriteFile(keyPath,
		pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER}), 0o600))

	signer, err := spoke_ca.Load(certPath, keyPath, 30)
	require.NoError(t, err)
	require.NotNil(t, signer)
	assert.Equal(t, "configured", signer.Source())

	signed, err := signer.Sign(makeCSR(t, "site-from-disk"), "")
	require.NoError(t, err)
	assert.Contains(t, signed.SubjectDN, "site-from-disk")
}
