package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aegis-imaging/aegis/api/middleware"
	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/tenantctx"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Audit log isolation tests. Every action a tenant takes is written with
// that tenant's tenant_id; the listing endpoints filter by tenant
// context so cross-tenant rows are invisible. Legacy single-tenant
// callers (no tenant context) see everything they would have seen
// pre-rollout.

func TestCreateAuditEntry_RecordsTenantIDFromContext(t *testing.T) {
	db := testutil.TestDB(t)

	tenant := &model.Tenant{Slug: "audit-write", Name: "Audit Write"}
	require.NoError(t, model.CreateTenant(context.Background(), db, tenant))

	ctx := tenantctx.With(context.Background(), tenant)
	require.NoError(t, model.CreateAuditEntry(ctx, db, "test.action", "user@x", "study", "study-1", "127.0.0.1", nil))

	// Look up the row directly.
	entries, err := model.ListAuditEntries(context.Background(), db, model.AuditFilters{Actor: "user@x"}, 10, 0)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.NotNil(t, entries[0].TenantID, "audit entry must carry tenant_id")
	assert.Equal(t, tenant.ID, *entries[0].TenantID)
}

func TestCreateAuditEntry_NilTenantWhenContextHasNoTenant(t *testing.T) {
	db := testutil.TestDB(t)

	require.NoError(t, model.CreateAuditEntry(context.Background(), db, "test.action", "legacy@x", "study", "study-1", "127.0.0.1", nil))

	entries, err := model.ListAuditEntries(context.Background(), db, model.AuditFilters{Actor: "legacy@x"}, 10, 0)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Nil(t, entries[0].TenantID, "legacy entry (no tenant in ctx) must store NULL tenant_id")
}

func TestListAudit_TenantHeaderFiltersByTenant(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	tenantA := &model.Tenant{Slug: "audit-a", Name: "A"}
	tenantB := &model.Tenant{Slug: "audit-b", Name: "B"}
	require.NoError(t, model.CreateTenant(context.Background(), db, tenantA))
	require.NoError(t, model.CreateTenant(context.Background(), db, tenantB))

	// 2 audit entries for A, 1 for B, 1 untenanted (legacy).
	ctxA := tenantctx.With(context.Background(), tenantA)
	ctxB := tenantctx.With(context.Background(), tenantB)
	require.NoError(t, model.CreateAuditEntry(ctxA, db, "tenant-a.event", "u@a", "x", "1", "1.1.1.1", nil))
	require.NoError(t, model.CreateAuditEntry(ctxA, db, "tenant-a.event2", "u@a", "x", "2", "1.1.1.1", nil))
	require.NoError(t, model.CreateAuditEntry(ctxB, db, "tenant-b.event", "u@b", "x", "1", "1.1.1.1", nil))
	require.NoError(t, model.CreateAuditEntry(context.Background(), db, "legacy.event", "u@legacy", "x", "1", "1.1.1.1", nil))

	req := httptest.NewRequest("GET", "/api/audit", nil)
	req.Header.Set(middleware.TenantHeader, "audit-a")
	rr := httptest.NewRecorder()
	middleware.ResolveTenant(db)(http.HandlerFunc(srv.ListAudit)).ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	var resp struct {
		Entries []model.AuditEntry `json:"entries"`
		Total   int                `json:"total"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, 2, resp.Total, "tenant A should see only its 2 entries")
	for _, e := range resp.Entries {
		require.NotNil(t, e.TenantID)
		assert.Equal(t, tenantA.ID, *e.TenantID)
	}
}

func TestListAudit_NoTenantHeaderShowsEverything(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	tenant := &model.Tenant{Slug: "audit-mixed", Name: "Mixed"}
	require.NoError(t, model.CreateTenant(context.Background(), db, tenant))
	ctxT := tenantctx.With(context.Background(), tenant)

	require.NoError(t, model.CreateAuditEntry(ctxT, db, "tenanted.event", "u@t", "x", "1", "1.1.1.1", nil))
	require.NoError(t, model.CreateAuditEntry(context.Background(), db, "legacy.event", "u@l", "x", "1", "1.1.1.1", nil))

	req := httptest.NewRequest("GET", "/api/audit", nil) // no tenant header
	rr := httptest.NewRecorder()
	middleware.ResolveTenant(db)(http.HandlerFunc(srv.ListAudit)).ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	var resp struct {
		Total int `json:"total"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.GreaterOrEqual(t, resp.Total, 2,
		"legacy single-tenant caller sees both tenanted and untenanted entries")
}

func TestListAudit_TenantNeverSeesUntenantedLegacyEntries(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	tenant := &model.Tenant{Slug: "audit-strict", Name: "Strict"}
	require.NoError(t, model.CreateTenant(context.Background(), db, tenant))

	// Legacy untenanted entry that the tenant must not see.
	require.NoError(t, model.CreateAuditEntry(context.Background(), db, "legacy.event", "u@legacy", "x", "1", "1.1.1.1", nil))

	req := httptest.NewRequest("GET", "/api/audit", nil)
	req.Header.Set(middleware.TenantHeader, "audit-strict")
	rr := httptest.NewRecorder()
	middleware.ResolveTenant(db)(http.HandlerFunc(srv.ListAudit)).ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	var resp struct {
		Entries []model.AuditEntry `json:"entries"`
		Total   int                `json:"total"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, 0, resp.Total,
		"a tenanted caller must not see legacy untenanted audit entries (defense in depth)")
}
