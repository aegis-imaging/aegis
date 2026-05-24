package middleware

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/tenantctx"
)

// Tenant-resolution middleware. SCAFFOLDING ONLY.
//
// This middleware *resolves* a tenant for every request that carries one,
// but does NOT yet enforce tenant-scoped access on downstream queries —
// that comes in subsequent PRs as each table grows a tenant_id column.
//
// Resolution order:
//  1. Explicit header `X-AEGIS-Tenant: <slug>` — used by service-to-service
//     callers, the desktop app, and the MCP server.
//  2. Subdomain on the Host header — `<slug>.api.aegisimaging.ai` resolves
//     to the tenant with that slug. We strip a configurable known suffix
//     (`api.aegisimaging.ai`, `aegis.<institution>.local`, etc.) and use
//     the remaining label as the slug.
//  3. Nothing — the request resolves to the "no tenant" state, which is
//     today identical to single-tenant behavior and is what every
//     existing legacy deployment needs.
//
// A resolved tenant is stored in the request context and retrievable via
// TenantFromContext. Handlers that need tenant scoping read it from there.

// TenantHeader is the explicit header callers set to choose a tenant.
const TenantHeader = "X-AEGIS-Tenant"

// TenantHostSuffixes is the list of host suffixes the middleware strips
// when looking for a subdomain slug. Configurable so on-prem deployments
// can register their own. Mutating from anywhere other than process
// startup is not supported.
//
// Note: `aegisimaging.ai` is deliberately NOT in the default list. It would
// match `api.aegisimaging.ai`, `admin.aegisimaging.ai`, `aws.api...`, etc.
// and treat the service name as a tenant slug — breaking every request to
// the canonical service hosts. Tenant subdomains live under
// `*.api.aegisimaging.ai` (e.g. `acme.api.aegisimaging.ai`).
var TenantHostSuffixes = []string{
	"api.aegisimaging.ai",
}

// ReservedTenantSlugs are labels that look like valid tenant slugs but are
// actually reserved service hostnames or infrastructure prefixes. The
// middleware always treats a request whose extracted candidate matches one
// of these as "no tenant" — even if the operator misconfigures
// TenantHostSuffixes to be too broad.
var ReservedTenantSlugs = map[string]bool{
	"api":    true,
	"admin":  true,
	"upload": true,
	"export": true,
	"landing": true,
	"www":    true,
	"aws":    true, // aws.api.aegisimaging.ai — AWS region of the service
	"gcp":    true,
	"azure":  true,
	"eu":     true, // future per-region prefixes
	"us":     true,
}

// ErrTenantDisabled is returned when a request resolves a tenant slug
// that exists but is disabled. Distinct from "not found" so callers can
// log it.
var ErrTenantDisabled = errors.New("tenant is disabled")

// ResolveTenant returns a middleware that runs tenant resolution on each
// request. db is the application DB pool; both arguments may be nil in
// tests that don't exercise tenant resolution (the middleware then no-ops).
//
// On success — including the "no tenant" success case — the next handler
// is called. On a slug that points at a disabled or missing tenant, we
// fail closed with HTTP 404; calling code shouldn't be making decisions
// based on the absence of a tenant when the caller explicitly named one.
func ResolveTenant(db *sql.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if db == nil {
				next.ServeHTTP(w, r)
				return
			}

			slug := tenantSlugFromRequest(r)
			if slug == "" {
				// No-tenant request — legacy single-tenant deployments hit
				// this path. The handler chain runs as normal.
				next.ServeHTTP(w, r)
				return
			}

			tenant, err := model.GetTenantBySlug(r.Context(), db, slug)
			if err != nil {
				// Both "not found" and "disabled" surface as 404 here so
				// we don't leak whether a tenant slug exists but is off.
				http.Error(w, "tenant not found", http.StatusNotFound)
				return
			}

			next.ServeHTTP(w, r.WithContext(tenantctx.With(r.Context(), tenant)))
		})
	}
}

// TenantFromContext returns the resolved tenant for a request, or nil
// when the request is in legacy single-tenant mode (no slug at all).
//
// Thin re-export of tenantctx.From — handlers were already importing
// middleware, so this keeps the existing call sites compiling. New
// callers in non-middleware-importing packages (like the model layer)
// should call tenantctx.From / tenantctx.ID directly.
func TenantFromContext(ctx context.Context) *model.Tenant {
	t, _ := tenantctx.From(ctx).(*model.Tenant)
	return t
}

// tenantSlugFromRequest pulls the slug out of either the explicit header
// or the Host subdomain. Returns "" when neither is present.
func tenantSlugFromRequest(r *http.Request) string {
	// 1. Explicit header wins.
	if v := strings.TrimSpace(r.Header.Get(TenantHeader)); v != "" {
		return strings.ToLower(v)
	}
	// 2. Subdomain. r.Host can include a :port suffix; strip it.
	host := r.Host
	if i := strings.IndexByte(host, ':'); i >= 0 {
		host = host[:i]
	}
	host = strings.ToLower(host)
	for _, suffix := range TenantHostSuffixes {
		dotSuffix := "." + suffix
		if strings.HasSuffix(host, dotSuffix) {
			candidate := strings.TrimSuffix(host, dotSuffix)
			// Reject candidates with internal dots — `foo.bar.api.aegisimaging.ai`
			// is two levels deep and shouldn't resolve to a tenant.
			if candidate == "" || strings.ContainsRune(candidate, '.') {
				continue
			}
			// Reject reserved service / region labels. Without this, a
			// request to `aws.api.aegisimaging.ai/healthz` would be
			// interpreted as tenant slug "aws", and unless a real
			// "aws"-named tenant exists the middleware 404s the request —
			// breaking infra-level smoke checks and any inter-region
			// routing.
			if ReservedTenantSlugs[candidate] {
				continue
			}
			return candidate
		}
	}
	return ""
}
