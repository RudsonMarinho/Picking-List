CREATE TABLE tenants (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    nome       VARCHAR(255) NOT NULL,
    slug       VARCHAR(255) UNIQUE NOT NULL,
    status     VARCHAR(50) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE plans (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    codigo     VARCHAR(50) UNIQUE NOT NULL,
    limites    JSONB NOT NULL,     -- max_assets, max_users, max_branches
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE subscriptions (
    tenant_id          UUID NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    plan_id            UUID NOT NULL REFERENCES plans(id)   ON DELETE RESTRICT,
    status             VARCHAR(50) NOT NULL,
    current_period_end TIMESTAMPTZ,
    provider           VARCHAR(100) NULL,
    provider_ref       VARCHAR(255) NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id)
);

CREATE TABLE billing_events (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    event_type VARCHAR(100) NOT NULL,
    payload    JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX ix_billing_events_tenant ON billing_events(tenant_id);
