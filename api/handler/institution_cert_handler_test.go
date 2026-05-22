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
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mintPEM(t *testing.T, cn string) string {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	tpl := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: cn},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, &key.PublicKey, key)
	require.NoError(t, err)
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
}

func TestSetInstitutionClientCert_FromPEM(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	inst := testutil.CreateTestInstitution(t, db, "cert-pem")

	body, _ := json.Marshal(map[string]string{"cert_pem": mintPEM(t, "cert-pem")})
	req := httptest.NewRequest(http.MethodPut, "/api/institutions/"+inst.ID+"/client-cert", strings.NewReader(string(body)))
	req.SetPathValue("id", inst.ID)
	rr := httptest.NewRecorder()
	srv.SetInstitutionClientCert(rr, req)
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

	var resp map[string]string
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Len(t, resp["thumbprint"], 64) // SHA-256 hex == 64 chars
	assert.Contains(t, resp["subject_dn"], "cert-pem")

	got, err := model.GetInstitutionByClientCertThumbprint(context.Background(), db, resp["thumbprint"])
	require.NoError(t, err)
	assert.Equal(t, inst.ID, got.ID)
}

func TestSetInstitutionClientCert_FromThumbprint(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	inst := testutil.CreateTestInstitution(t, db, "cert-thumb")

	body := `{"thumbprint":"aa:bb:cc:dd:ee:ff:00:11:22:33:44:55:66:77:88:99:aa:bb:cc:dd:ee:ff:00:11:22:33:44:55:66:77:88:99"}`
	req := httptest.NewRequest(http.MethodPut, "/api/institutions/"+inst.ID+"/client-cert", strings.NewReader(body))
	req.SetPathValue("id", inst.ID)
	rr := httptest.NewRecorder()
	srv.SetInstitutionClientCert(rr, req)
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

	var resp map[string]string
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	// Colons stripped, lowercased.
	assert.Equal(t, "aabbccddeeff00112233445566778899aabbccddeeff00112233445566778899", resp["thumbprint"])
}

func TestSetInstitutionClientCert_RejectsEmpty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	inst := testutil.CreateTestInstitution(t, db, "cert-empty")

	req := httptest.NewRequest(http.MethodPut, "/api/institutions/"+inst.ID+"/client-cert", strings.NewReader(`{}`))
	req.SetPathValue("id", inst.ID)
	rr := httptest.NewRecorder()
	srv.SetInstitutionClientCert(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestRevokeInstitutionClientCert(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	inst := testutil.CreateTestInstitution(t, db, "cert-revoke")

	body, _ := json.Marshal(map[string]string{"cert_pem": mintPEM(t, "cert-revoke")})
	setReq := httptest.NewRequest(http.MethodPut, "/api/institutions/"+inst.ID+"/client-cert", strings.NewReader(string(body)))
	setReq.SetPathValue("id", inst.ID)
	setRR := httptest.NewRecorder()
	srv.SetInstitutionClientCert(setRR, setReq)
	require.Equal(t, http.StatusOK, setRR.Code)
	var setResp map[string]string
	require.NoError(t, json.NewDecoder(setRR.Body).Decode(&setResp))

	revReq := httptest.NewRequest(http.MethodDelete, "/api/institutions/"+inst.ID+"/client-cert", nil)
	revReq.SetPathValue("id", inst.ID)
	revRR := httptest.NewRecorder()
	srv.RevokeInstitutionClientCert(revRR, revReq)
	assert.Equal(t, http.StatusOK, revRR.Code)

	// Lookup by the previous thumbprint should fail post-revoke.
	_, err := model.GetInstitutionByClientCertThumbprint(context.Background(), db, setResp["thumbprint"])
	assert.Error(t, err)
}
