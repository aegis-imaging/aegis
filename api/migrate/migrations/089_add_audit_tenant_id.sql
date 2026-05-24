-- +goose Up
-- Multi-tenant rollout — chunk 4: audit_trail tenant scoping.
--
-- Audit log isolation is the most security-critical piece of the tenant
-- boundary: a tenant must not be able to see another tenant's actions even
-- via direct DB query. We denormalize tenant_id onto each audit row at
-- INSERT time (cheap — every CreateAuditEntry call already has request
-- context, which already carries the resolved tenant). Existing rows stay
-- NULL = "pre-tenant-rollout", visible to legacy single-tenant callers
-- only.
ALTER TABLE audit_trail
    ADD COLUMN IF NOT EXISTS tenant_id UUID REFERENCES tenants(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_audit_trail_tenant
    ON audit_trail(tenant_id, created_at DESC)
    WHERE tenant_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_audit_trail_tenant;
ALTER TABLE audit_trail DROP COLUMN IF EXISTS tenant_id;
