package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// White-box unit tests for the slug-parsing helper. Kept in
// `package middleware` so they can call the unexported
// `tenantSlugFromRequest` directly. They MUST NOT import
// `api/testutil` — that would create an import cycle
// (testutil → handler → middleware).
// DB-backed integration tests for `ResolveTenant` live in
// `tenant_integration_test.go` under `package middleware_test`.

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

// Regression: requests to the canonical api host MUST NOT be interpreted
// as tenant requests (slug="api"). Before this guard, the
// middleware 404'd every request to api.aegisimaging.ai because no
// tenant named "api" existed — breaking the post-deploy /healthz
// smoke check on both AWS and GCP.
func TestTenantSlugFromRequest_ServiceHostIsNotTenant(t *testing.T) {
	r := httptest.NewRequest("GET", "http://x/healthz", nil)
	r.Host = "api.aegisimaging.ai"
	assert.Equal(t, "", tenantSlugFromRequest(r),
		"api.aegisimaging.ai is the service host, not a tenant")
}

func TestTenantSlugFromRequest_RegionalServiceHostIsNotTenant(t *testing.T) {
	// aws.api.aegisimaging.ai is the AWS regional copy of the api host.
	// "aws" is in ReservedTenantSlugs so this must NOT resolve as a tenant.
	r := httptest.NewRequest("GET", "http://x/healthz", nil)
	r.Host = "aws.api.aegisimaging.ai"
	assert.Equal(t, "", tenantSlugFromRequest(r),
		"aws.api.aegisimaging.ai is the AWS regional service host, not a tenant")
}

func TestTenantSlugFromRequest_ReservedAdminLabel(t *testing.T) {
	r := httptest.NewRequest("GET", "http://x/", nil)
	r.Host = "admin.api.aegisimaging.ai"
	assert.Equal(t, "", tenantSlugFromRequest(r),
		"`admin` is reserved — should not be interpreted as a tenant slug")
}

func TestTenantSlugFromRequest_RealTenantUnderApiSuffix(t *testing.T) {
	// The legitimate tenant pattern: <tenant>.api.aegisimaging.ai.
	r := httptest.NewRequest("GET", "http://x/", nil)
	r.Host = "acme.api.aegisimaging.ai"
	assert.Equal(t, "acme", tenantSlugFromRequest(r),
		"non-reserved labels under api.aegisimaging.ai resolve as tenants")
}
