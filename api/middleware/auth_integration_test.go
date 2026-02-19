package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/msenjem/aegis/api/config"
	"github.com/msenjem/aegis/api/middleware"
	"github.com/msenjem/aegis/api/model"
	"github.com/msenjem/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequireAuth_DevMode(t *testing.T) {
	db := testutil.TestDB(t)
	cfg := &config.Config{AuthEnabled: false, DevUserEmail: "dev@aegis.local"}

	var capturedUser *middleware.AuthUser
	wrapped := middleware.RequireAuth(db, cfg)(func(w http.ResponseWriter, r *http.Request) {
		capturedUser = middleware.UserFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	wrapped(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	require.NotNil(t, capturedUser)
	assert.Equal(t, "dev@aegis.local", capturedUser.Email)
	assert.Equal(t, "admin", capturedUser.Role, "dev user should be admin by default")
}

func TestRequireAuth_DevMode_ExistingUser(t *testing.T) {
	db := testutil.TestDB(t)
	testutil.CreateTestAdminUser(t, db, "dev@aegis.local", "viewer")
	cfg := &config.Config{AuthEnabled: false, DevUserEmail: "dev@aegis.local"}

	var capturedUser *middleware.AuthUser
	wrapped := middleware.RequireAuth(db, cfg)(func(w http.ResponseWriter, r *http.Request) {
		capturedUser = middleware.UserFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	wrapped(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "viewer", capturedUser.Role, "should use DB role when user exists")
}

func TestRequireAuth_IAP(t *testing.T) {
	db := testutil.TestDB(t)
	model.CreateAdminUser(context.Background(), db, &model.AdminUser{
		Email: "user@example.com", Name: "Test", Role: "admin", Enabled: true,
	})
	cfg := &config.Config{AuthEnabled: true, AuthProvider: "iap"}

	var capturedUser *middleware.AuthUser
	wrapped := middleware.RequireAuth(db, cfg)(func(w http.ResponseWriter, r *http.Request) {
		capturedUser = middleware.UserFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Goog-Authenticated-User-Email", "accounts.google.com:User@Example.com")
	rr := httptest.NewRecorder()
	wrapped(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "user@example.com", capturedUser.Email)
}

func TestRequireAuth_Azure(t *testing.T) {
	db := testutil.TestDB(t)
	model.CreateAdminUser(context.Background(), db, &model.AdminUser{
		Email: "azure@example.com", Name: "Test", Role: "admin", Enabled: true,
	})
	cfg := &config.Config{AuthEnabled: true, AuthProvider: "azure"}

	var capturedUser *middleware.AuthUser
	wrapped := middleware.RequireAuth(db, cfg)(func(w http.ResponseWriter, r *http.Request) {
		capturedUser = middleware.UserFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-MS-CLIENT-PRINCIPAL-NAME", "Azure@Example.com")
	rr := httptest.NewRecorder()
	wrapped(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "azure@example.com", capturedUser.Email)
}

func TestRequireAuth_MissingHeader(t *testing.T) {
	db := testutil.TestDB(t)
	cfg := &config.Config{AuthEnabled: true, AuthProvider: "iap"}

	wrapped := middleware.RequireAuth(db, cfg)(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	wrapped(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestRequireAuth_UnknownUser(t *testing.T) {
	db := testutil.TestDB(t)
	cfg := &config.Config{AuthEnabled: true, AuthProvider: "iap"}

	wrapped := middleware.RequireAuth(db, cfg)(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Goog-Authenticated-User-Email", "accounts.google.com:unknown@test.com")
	rr := httptest.NewRecorder()
	wrapped(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestRequireAuth_DisabledUser(t *testing.T) {
	db := testutil.TestDB(t)
	u := &model.AdminUser{
		Email: "disabled@test.com", Name: "Disabled", Role: "admin", Enabled: false,
	}
	model.CreateAdminUser(context.Background(), db, u)
	cfg := &config.Config{AuthEnabled: true, AuthProvider: "iap"}

	wrapped := middleware.RequireAuth(db, cfg)(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Goog-Authenticated-User-Email", "accounts.google.com:disabled@test.com")
	rr := httptest.NewRecorder()
	wrapped(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestRequireRole_AdminAllowed(t *testing.T) {
	db := testutil.TestDB(t)
	model.CreateAdminUser(context.Background(), db, &model.AdminUser{
		Email: "admin@test.com", Name: "Admin", Role: "admin", Enabled: true,
	})
	cfg := &config.Config{AuthEnabled: true, AuthProvider: "iap"}

	wrapped := middleware.RequireRole("admin", db, cfg)(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("X-Goog-Authenticated-User-Email", "accounts.google.com:admin@test.com")
	rr := httptest.NewRecorder()
	wrapped(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRequireRole_ViewerDenied(t *testing.T) {
	db := testutil.TestDB(t)
	model.CreateAdminUser(context.Background(), db, &model.AdminUser{
		Email: "viewer@test.com", Name: "Viewer", Role: "viewer", Enabled: true,
	})
	cfg := &config.Config{AuthEnabled: true, AuthProvider: "iap"}

	wrapped := middleware.RequireRole("admin", db, cfg)(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("X-Goog-Authenticated-User-Email", "accounts.google.com:viewer@test.com")
	rr := httptest.NewRecorder()
	wrapped(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}
