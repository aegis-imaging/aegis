package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTenantSlugFromRequest_Header(t *testing.T) {
	r := httptest.NewRequest("GET", "http://example.com/x", nil)
	r.Header.Set(TenantHeader, "acme")
	assert.Equal(t, "acme", tenantSlugFromRequest(r))
}

func TestTenantSlugFromRequest_HeaderLowercased(t *testing.T) {
	r := httptest.NewRequest("GET", "http://example.com/x", nil)
	r.Header.Set(TenantHeader, " ACME ")
	assert.Equal(t, "acme", tenantSlugFromRequest(r))
}

func TestTenantSlugFromRequest_SubdomainOfKnownSuffix(t *testing.T) {
	r := httptest.NewRequest("GET", "http://x.api.aegisimaging.ai/y", nil)
	r.Host = "acme.api.aegisimaging.ai"
	assert.Equal(t, "acme", tenantSlugFromRequest(r))
}

func TestTenantSlugFromRequest_SubdomainStripsPort(t *testing.T) {
	r := httptest.NewRequest("GET", "http://x/y", nil)
	r.Host = "acme.api.aegisimaging.ai:8080"
	assert.Equal(t, "acme", tenantSlugFromRequest(r))
}

func TestTenantSlugFromRequest_HeaderBeatsSubdomain(t *testing.T) {
	r := httptest.NewRequest("GET", "http://x/y", nil)
	r.Host = "subdomain.api.aegisimaging.ai"
	r.Header.Set(TenantHeader, "header-wins")
	assert.Equal(t, "header-wins", tenantSlugFromRequest(r))
}

func TestTenantSlugFromRequest_RejectsNestedSubdomain(t *testing.T) {
	// `eu.acme.api.aegisimaging.ai` is two labels deep — that's the
	// region-then-tenant style and shouldn't resolve to a tenant with
	// our naive rule. Callers using nested labels need the explicit
	// header.
	r := httptest.NewRequest("GET", "http://x/y", nil)
	r.Host = "eu.acme.api.aegisimaging.ai"
	assert.Equal(t, "", tenantSlugFromRequest(r))
}

func TestTenantSlugFromRequest_NoMatch(t *testing.T) {
	r := httptest.NewRequest("GET", "http://localhost/", nil)
	r.Host = "localhost"
	assert.Equal(t, "", tenantSlugFromRequest(r))
}

func TestResolveTenant_NoSlugMeansNoTenantContext(t *testing.T) {
	db := testutil.TestDB(t)

	var sawTenant *model.Tenant
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawTenant = TenantFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	r := httptest.NewRequest("GET", "http://localhost/api/healthz", nil)
	rr := httptest.NewRecorder()
	ResolveTenant(db)(next).ServeHTTP(rr, r)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Nil(t, sawTenant, "no slug → no tenant in context (legacy single-tenant request)")
}

func TestResolveTenant_HeaderResolvesAndStoresInContext(t *testing.T) {
	db := testutil.TestDB(t)

	// Create a tenant directly via the model.
	tenant := &model.Tenant{Slug: "acme", Name: "Acme Co"}
	require.NoError(t, model.CreateTenant(context.Background(), db, tenant))

	var seen *model.Tenant
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = TenantFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	r := httptest.NewRequest("GET", "http://localhost/api/foo", nil)
	r.Header.Set(TenantHeader, "acme")
	rr := httptest.NewRecorder()
	ResolveTenant(db)(next).ServeHTTP(rr, r)

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
	r.Header.Set(TenantHeader, "no-such-tenant")
	rr := httptest.NewRecorder()
	ResolveTenant(db)(next).ServeHTTP(rr, r)

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
	r.Header.Set(TenantHeader, "old-tenant")
	rr := httptest.NewRecorder()
	ResolveTenant(db)(next).ServeHTTP(rr, r)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.False(t, called, "disabled tenants are invisible — same 404 as a missing slug")
}

func TestResolveTenant_NilDBNoOps(t *testing.T) {
	// When the application boots without a DB (some test paths do this),
	// the middleware should be transparent.
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Nil(t, TenantFromContext(r.Context()))
		w.WriteHeader(http.StatusTeapot)
	})

	r := httptest.NewRequest("GET", "http://localhost/", nil)
	r.Header.Set(TenantHeader, "anything")
	rr := httptest.NewRecorder()
	ResolveTenant(nil)(next).ServeHTTP(rr, r)

	assert.Equal(t, http.StatusTeapot, rr.Code)
}
