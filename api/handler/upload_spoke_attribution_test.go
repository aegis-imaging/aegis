package handler_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aegis-imaging/aegis/api/middleware"
	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// withSpokeCtx puts a SpokeIdentity onto the request context the same way the
// middleware would, but without depending on header parsing — tests can then
// drive the handler directly.
func withSpokeCtx(r *http.Request, inst *model.Institution) *http.Request {
	ident := &middleware.SpokeIdentity{
		Institution:    inst,
		CertThumbprint: "deadbeef",
		CertSubjectDN:  "CN=spoke-test",
	}
	ctx := context.WithValue(r.Context(), middleware.SpokeIdentityContextKey(), ident)
	return r.WithContext(ctx)
}

func TestUploadInit_SpokeIdentityOverridesAttribution(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	// Two institutions: one whose ID is in the request body, and another that
	// owns the spoke cert. The spoke cert must win.
	spokeInst := testutil.CreateTestInstitution(t, db, "spoke-attribution-spoke")
	bodyInst := testutil.CreateTestInstitution(t, db, "spoke-attribution-body")
	// Spoke institutions need sender capability + a project link.
	proj := testutil.SeedProject(t, db)
	require.NoError(t, model.AddInstitutionToProject(context.Background(), db, &model.InstitutionProject{
		InstitutionID: spokeInst.ID, ProjectID: proj.ID, Role: "sender",
	}))

	body, _ := json.Marshal(map[string]any{
		"project_slug":   "default",
		"institution_id": bodyInst.ID,
		"file_count":     2,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/upload/init", strings.NewReader(string(body)))
	req = withSpokeCtx(req, spokeInst)
	rr := httptest.NewRecorder()
	srv.UploadInit(rr, req)
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

	var resp struct {
		SessionID string `json:"session_id"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	require.NotEmpty(t, resp.SessionID)

	session, err := model.GetUploadSession(context.Background(), db, resp.SessionID)
	require.NoError(t, err)
	require.NotNil(t, session.InstitutionID, "spoke should have attributed an institution")
	assert.Equal(t, spokeInst.ID, *session.InstitutionID, "spoke cert must win over body institution_id")
}

func TestUploadInit_NoSpokeKeepsExistingBehavior(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	body, _ := json.Marshal(map[string]any{
		"project_slug": "default",
		"file_count":   1,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/upload/init", strings.NewReader(string(body)))
	rr := httptest.NewRecorder()
	srv.UploadInit(rr, req)
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
}

// mintRealCert produces an actual self-signed cert + computes its thumbprint
// so the middleware integration tests below verify the end-to-end identity
// match (not just the context-injection shortcut used in attribution tests).
func mintRealCert(t *testing.T, cn string) (thumb, subj string) {
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
	sum := sha256.Sum256(der)
	thumb = hex.EncodeToString(sum[:])
	cert, _ := x509.ParseCertificate(der)
	subj = cert.Subject.String()
	return
}

func TestUploadInit_RealEnrolledCertAttributes(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	inst := testutil.CreateTestInstitution(t, db, "spoke-real-cert")
	proj := testutil.SeedProject(t, db)
	require.NoError(t, model.AddInstitutionToProject(context.Background(), db, &model.InstitutionProject{
		InstitutionID: inst.ID, ProjectID: proj.ID, Role: "sender",
	}))

	thumb, subj := mintRealCert(t, "spoke-real-cert")
	require.NoError(t, model.SetInstitutionClientCert(context.Background(), db, inst.ID, thumb, subj))

	// Drive the request through the actual middleware so we exercise the
	// thumbprint lookup path end-to-end (not just SpokeFromContext).
	body, _ := json.Marshal(map[string]any{"project_slug": "default", "file_count": 1})
	req := httptest.NewRequest(http.MethodPost, "/api/upload/init", strings.NewReader(string(body)))
	// Use the fingerprint header form so we don't need the full PEM.
	req.Header.Set("X-Client-Cert-Fingerprint", thumb)

	wrapped := middleware.WithSpokeMTLS(db)(srv.UploadInit)
	rr := httptest.NewRecorder()
	wrapped(rr, req)
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

	var resp struct {
		SessionID string `json:"session_id"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	session, err := model.GetUploadSession(context.Background(), db, resp.SessionID)
	require.NoError(t, err)
	require.NotNil(t, session.InstitutionID)
	assert.Equal(t, inst.ID, *session.InstitutionID)
}
