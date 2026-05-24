// Package tenantctx carries the resolved multi-tenant request context
// across the request lifecycle.
//
// It lives in its own (tiny) package so both the middleware that resolves
// the tenant from the request and the model layer that needs to attach
// tenant_id to writes can read/write the same context value without
// creating an import cycle (middleware already imports model, so model
// can't import middleware to call middleware.TenantFromContext).
//
// Production code path:
//   1. middleware.ResolveTenant(db) resolves the tenant from the X-AEGIS-Tenant
//      header (or subdomain) and stores it via tenantctx.With(ctx, t).
//   2. handler code reads it via tenantctx.From(r.Context()).
//   3. model code (e.g. CreateAuditEntry) also reads it via tenantctx.From,
//      so audit writes inherit the tenant boundary without every handler
//      threading it through explicitly.
package tenantctx

import (
	"context"

	"github.com/aegis-imaging/aegis/api/model"
)

// key is the context-value type — unexported so callers can't accidentally
// overwrite it with a string of the same value.
type key struct{}

var tenantKey = key{}

// With returns a derived context that carries the given tenant. Passing
// nil is allowed and is equivalent to "no tenant" (legacy single-tenant
// request) — but consider just not calling With at all in that case.
func With(ctx context.Context, t *model.Tenant) context.Context {
	return context.WithValue(ctx, tenantKey, t)
}

// From returns the tenant resolved for this request, or nil for a legacy
// single-tenant request (no header, no matching subdomain).
func From(ctx context.Context) *model.Tenant {
	t, _ := ctx.Value(tenantKey).(*model.Tenant)
	return t
}

// ID returns the tenant ID from the context, or "" if no tenant is set.
// Convenience for callers that just want to populate a tenant_id column
// without nil-checking the *Tenant pointer.
func ID(ctx context.Context) string {
	if t := From(ctx); t != nil {
		return t.ID
	}
	return ""
}
