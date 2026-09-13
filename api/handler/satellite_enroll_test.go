package handler_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/satellite_ca"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeSatelliteCSR(t *testing.T, cn string) string {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	der, err := x509.CreateCertificateRequest(rand.Reader,
		&x509.CertificateRequest{Subject: pkix.Name{CommonName: cn}}, key)
	require.NoError(t, err)
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: der}))
}

func TestEnrollSatellite_HappyPath(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	signer, err := satellite_ca.NewEphemeral(30)
	require.NoError(t, err)
	srv.SetSatelliteCA(signer)

	inst := testutil.CreateTestInstitution(t, db, "enroll-real-spoke")

	raw, hash, err := model.GenerateEnrollmentToken()
	require.NoError(t, err)
	tok := &model.SatelliteEnrollmentToken{
		InstitutionID: inst.ID, CreatedBy: "admin@test.com",
		ExpiresAt: time.Now().UTC().Add(1 * time.Hour),
	}
	require.NoError(t, model.CreateSatelliteEnrollmentToken(context.Background(), db, tok, hash))

	body, _ := json.Marshal(map[string]string{
		"token":   raw,
		"csr_pem": makeSatelliteCSR(t, "umn-lab-satellite"),
		"site_id": "umn-lab-satellite",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/satellites/enroll", strings.NewReader(string(body)))
	rr := httptest.NewRecorder()
	srv.EnrollSatellite(rr, req)
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

	var resp struct {
		ClientCertPEM string `json:"client_cert_pem"`
		CAPEM         string `json:"ca_pem"`
		Thumbprint    string `json:"thumbprint"`
		SubjectDN     string `json:"subject_dn"`
		InstitutionID string `json:"institution_id"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Contains(t, resp.ClientCertPEM, "BEGIN CERTIFICATE")
	assert.Contains(t, resp.CAPEM, "BEGIN CERTIFICATE")
	assert.Len(t, resp.Thumbprint, 64)
	assert.Equal(t, inst.ID, resp.InstitutionID)

	// Verify the institution now resolves by thumbprint (i.e. mTLS middleware
	// will accept this cert on subsequent uploads).
	resolved, err := model.GetInstitutionByClientCertThumbprint(context.Background(), db, resp.Thumbprint)
	require.NoError(t, err)
	assert.Equal(t, inst.ID, resolved.ID)
}

func TestEnrollSatellite_DisabledReturns503(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	// No SetSatelliteCA called — satelliteCA stays nil.

	body, _ := json.Marshal(map[string]string{"token": "x", "csr_pem": makeSatelliteCSR(t, "x")})
	req := httptest.NewRequest(http.MethodPost, "/api/satellites/enroll", strings.NewReader(string(body)))
	rr := httptest.NewRecorder()
	srv.EnrollSatellite(rr, req)
	assert.Equal(t, http.StatusServiceUnavailable, rr.Code)
}

func TestEnrollSatellite_InvalidTokenReturns401(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	signer, _ := satellite_ca.NewEphemeral(30)
	srv.SetSatelliteCA(signer)

	body, _ := json.Marshal(map[string]string{
		"token": "never-issued", "csr_pem": makeSatelliteCSR(t, "x"),
	})
	req := httptest.NewRequest(http.MethodPost, "/api/satellites/enroll", strings.NewReader(string(body)))
	rr := httptest.NewRecorder()
	srv.EnrollSatellite(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestEnrollSatellite_TokenSingleUse(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	signer, _ := satellite_ca.NewEphemeral(30)
	srv.SetSatelliteCA(signer)

	inst := testutil.CreateTestInstitution(t, db, "enroll-single-use")
	raw, hash, _ := model.GenerateEnrollmentToken()
	tok := &model.SatelliteEnrollmentToken{InstitutionID: inst.ID, ExpiresAt: time.Now().UTC().Add(1 * time.Hour)}
	require.NoError(t, model.CreateSatelliteEnrollmentToken(context.Background(), db, tok, hash))

	body, _ := json.Marshal(map[string]string{"token": raw, "csr_pem": makeSatelliteCSR(t, "x")})
	req := httptest.NewRequest(http.MethodPost, "/api/satellites/enroll", strings.NewReader(string(body)))
	rr := httptest.NewRecorder()
	srv.EnrollSatellite(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	// Second redemption with the same token must fail.
	req2 := httptest.NewRequest(http.MethodPost, "/api/satellites/enroll", strings.NewReader(string(body)))
	rr2 := httptest.NewRecorder()
	srv.EnrollSatellite(rr2, req2)
	assert.Equal(t, http.StatusUnauthorized, rr2.Code)
}

func TestEnrollSatellite_InvalidCSR(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	signer, _ := satellite_ca.NewEphemeral(30)
	srv.SetSatelliteCA(signer)
	inst := testutil.CreateTestInstitution(t, db, "enroll-bad-csr")
	raw, hash, _ := model.GenerateEnrollmentToken()
	tok := &model.SatelliteEnrollmentToken{InstitutionID: inst.ID, ExpiresAt: time.Now().UTC().Add(1 * time.Hour)}
	require.NoError(t, model.CreateSatelliteEnrollmentToken(context.Background(), db, tok, hash))

	body, _ := json.Marshal(map[string]string{"token": raw, "csr_pem": "not a CSR"})
	req := httptest.NewRequest(http.MethodPost, "/api/satellites/enroll", strings.NewReader(string(body)))
	rr := httptest.NewRecorder()
	srv.EnrollSatellite(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}
