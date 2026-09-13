// Package tenantctx carries the resolved multi-tenant request context
// across the request lifecycle.
//
// It lives in its own (tiny) package so both the middleware that resolves
// the tenant from the request and the model layer that needs to attach
// tenant_id to writes can read/write the same context value without
// creating an import cycle.
//
// To keep this package leaf-level (no api/model import), the stored value
// is `any`. Consumers in packages that already import api/model type-assert
// to *model.Tenant; the model layer reaches the tenant ID via the small
// duck-typed interface below (model.Tenant satisfies it).
//
// Production code path:
//  1. middleware.ResolveTenant(db) resolves the tenant from the X-AEGIS-Tenant
//     header (or subdomain) and stores it via tenantctx.With(ctx, t).
//  2. handler code reads it via middleware.TenantFromContext (which
//     type-asserts back to *model.Tenant).
//  3. model code (e.g. CreateAuditEntry) reads the ID via tenantctx.ID,
//     so audit writes inherit the tenant boundary without every handler
//     threading it through explicitly.
package tenantctx

import "context"

// key is the context-value type — unexported so callers can't accidentally
// overwrite it with a string of the same value.
type key struct{}

var tenantKey = key{}

// idGetter is the duck-typed contract the stored tenant value must satisfy
// for ID() to extract its tenant id. *model.Tenant satisfies this via its
// TenantID() method (defined alongside the struct in api/model).
type idGetter interface {
	TenantID() string
}

// With returns a derived context that carries the given tenant. The value
// is stored as `any` to keep this package free of an api/model import.
// Passing nil is allowed and is equivalent to "no tenant" (legacy
// single-tenant request) — but consider just not calling With at all in
// that case.
func With(ctx context.Context, t any) context.Context {
	return context.WithValue(ctx, tenantKey, t)
}

// From returns the raw tenant value stored on the context, or nil if none
// is set. Callers that need *model.Tenant should type-assert (or use the
// middleware.TenantFromContext helper, which does the assertion).
func From(ctx context.Context) any {
	return ctx.Value(tenantKey)
}

// ID returns the tenant ID from the context, or "" if no tenant is set
// (or the stored value does not satisfy idGetter). Convenience for the
// model layer, which only needs the ID column and can't reach back into
// api/model from here.
func ID(ctx context.Context) string {
	if t, ok := From(ctx).(idGetter); ok && t != nil {
		return t.TenantID()
	}
	return ""
}
