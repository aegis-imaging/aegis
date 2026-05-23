-- +goose Up
-- Multi-tenant SaaS scaffolding (chunk 1).
--
-- A "tenant" is the top-level isolation boundary for a future SaaS
-- deployment — one tenant per customer organisation. Projects (and,
-- in follow-up PRs, their child rows) carry a tenant_id so cross-tenant
-- queries can be blocked at the model layer.
--
-- This migration is intentionally NOT-NULL-free: existing rows in
-- single-tenant deployments stay tenant_id IS NULL and the API still
-- works exactly as before. Tenant resolution + enforcement is added
-- incrementally in subsequent PRs.
CREATE TABLE tenants (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug        TEXT NOT NULL UNIQUE,                              -- URL-safe; used in subdomain and X-AEGIS-Tenant header
    name        TEXT NOT NULL,
    settings    JSONB NOT NULL DEFAULT '{}'::jsonb,                -- tenant-scoped configuration (rate limits, feature flags, etc.)
    enabled     BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tenants_slug_enabled ON tenants(slug) WHERE enabled = TRUE;

-- Projects opt into a tenant when populated. NULL = legacy single-tenant
-- deployment; the API treats those projects as belonging to no tenant
-- (which today is identical to belonging to "the implicit default").
ALTER TABLE projects ADD COLUMN tenant_id UUID REFERENCES tenants(id) ON DELETE RESTRICT;
CREATE INDEX idx_projects_tenant ON projects(tenant_id);

-- +goose Down
ALTER TABLE projects DROP COLUMN IF EXISTS tenant_id;
DROP TABLE IF EXISTS tenants;
