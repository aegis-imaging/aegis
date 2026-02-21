package middleware_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aegis-imaging/aegis/api/config"
	"github.com/aegis-imaging/aegis/api/middleware"
	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
)

func makeALBJWT(email string) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"ES256"}`))
	payloadJSON, _ := json.Marshal(map[string]string{"email": email})
	payload := base64.RawURLEncoding.EncodeToString(payloadJSON)
	return header + "." + payload + ".signature"
}

func TestRequireAuth_AWSProvider(t *testing.T) {
	db := testutil.TestDB(t)
	_ = model.CreateAdminUser(context.Background(), db, &model.AdminUser{
		Email: "aws-user@example.com", Name: "AWS User", Role: "admin", Enabled: true,
	})

	cfg := &config.Config{AuthEnabled: true, AuthProvider: "aws"}

	var capturedUser *middleware.AuthUser
	wrapped := middleware.RequireAuth(db, cfg)(func(w http.ResponseWriter, r *http.Request) {
		capturedUser = middleware.UserFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Amzn-Oidc-Data", makeALBJWT("AWS-USER@Example.com"))
	rr := httptest.NewRecorder()
	wrapped(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	if assert.NotNil(t, capturedUser) {
		assert.Equal(t, "aws-user@example.com", capturedUser.Email)
		assert.Equal(t, "admin", capturedUser.Role)
	}
}

func TestRequireAuth_AWSProviderMissingHeader(t *testing.T) {
	db := testutil.TestDB(t)
	cfg := &config.Config{AuthEnabled: true, AuthProvider: "aws"}

	wrapped := middleware.RequireAuth(db, cfg)(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()
	wrapped(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Contains(t, rr.Body.String(), "missing AWS ALB authentication header")
}

func TestRequireAuth_AutoProviderWithAWSHeader(t *testing.T) {
	db := testutil.TestDB(t)
	_ = model.CreateAdminUser(context.Background(), db, &model.AdminUser{
		Email: "auto-aws@example.com", Name: "Auto AWS", Role: "viewer", Enabled: true,
	})

	cfg := &config.Config{AuthEnabled: true, AuthProvider: "auto"}

	var capturedUser *middleware.AuthUser
	wrapped := middleware.RequireAuth(db, cfg)(func(w http.ResponseWriter, r *http.Request) {
		capturedUser = middleware.UserFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Amzn-Oidc-Data", makeALBJWT("auto-aws@example.com"))
	rr := httptest.NewRecorder()
	wrapped(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	if assert.NotNil(t, capturedUser) {
		assert.Equal(t, "auto-aws@example.com", capturedUser.Email)
		assert.Equal(t, "viewer", capturedUser.Role)
	}
}
