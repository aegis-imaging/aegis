package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aegis-imaging/aegis/api/middleware"
	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Black-box integration tests for ResolveTenant. Live in
// `package middleware_test` so they can import `testutil` without
// triggering the testutil → handler → middleware import cycle.

func TestResolveTenant_NoSlugMeansNoTenantContext(t *testing.T) {
	db := testutil.TestDB(t)

	var sawTenant *model.Tenant
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawTenant = middleware.TenantFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	r := httptest.NewRequest("GET", "http://localhost/api/healthz", nil)
	rr := httptest.NewRecorder()
	middleware.ResolveTenant(db)(next).ServeHTTP(rr, r)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Nil(t, sawTenant, "no slug → no tenant in context (legacy single-tenant request)")
}

func TestResolveTenant_HeaderResolvesAndStoresInContext(t *testing.T) {
	db := testutil.TestDB(t)

	tenant := &model.Tenant{Slug: "acme", Name: "Acme Co"}
	require.NoError(t, model.CreateTenant(context.Background(), db, tenant))

	var seen *model.Tenant
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = middleware.TenantFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	r := httptest.NewRequest("GET", "http://localhost/api/foo", nil)
	r.Header.Set(middleware.TenantHeader, "acme")
	rr := httptest.NewRecorder()
	middleware.ResolveTenant(db)(next).ServeHTTP(rr, r)

	assert.Equal(t, http.StatusOK, rr.Code)
	require.NotNil(t, seen)
	assert.Equal(t, tenant.ID, seen.ID)
	assert.Equal(t, "acme", seen.Slug)
}

func TestResolveTenant_UnknownSlugReturns404(t *testing.T) {
	db := testutil.TestDB(t)

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	r := httptest.NewRequest("GET", "http://localhost/api/foo", nil)
	r.Header.Set(middleware.TenantHeader, "no-such-tenant")
	rr := httptest.NewRecorder()
	middleware.ResolveTenant(db)(next).ServeHTTP(rr, r)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.False(t, called, "next handler must not run when an unknown tenant is named")
}

func TestResolveTenant_DisabledSlugReturns404(t *testing.T) {
	db := testutil.TestDB(t)

	tenant := &model.Tenant{Slug: "old-tenant", Name: "Old Tenant", Enabled: false}
	require.NoError(t, model.CreateTenant(context.Background(), db, tenant))

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	r := httptest.NewRequest("GET", "http://localhost/api/foo", nil)
	r.Header.Set(middleware.TenantHeader, "old-tenant")
	rr := httptest.NewRecorder()
	middleware.ResolveTenant(db)(next).ServeHTTP(rr, r)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.False(t, called, "disabled tenants are invisible — same 404 as a missing slug")
}

func TestResolveTenant_NilDBNoOps(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Nil(t, middleware.TenantFromContext(r.Context()))
		w.WriteHeader(http.StatusTeapot)
	})

	r := httptest.NewRequest("GET", "http://localhost/", nil)
	r.Header.Set(middleware.TenantHeader, "anything")
	rr := httptest.NewRecorder()
	middleware.ResolveTenant(nil)(next).ServeHTTP(rr, r)

	assert.Equal(t, http.StatusTeapot, rr.Code)
}
