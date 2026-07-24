CREATE OR REPLACE FUNCTION set_updated_at() RETURNS trigger AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ---------- Lookups pré-tenant (SEM RLS) ----------
-- Resolvem o tenant ANTES do SET LOCAL. Contêm APENAS roteamento:
-- zero hash de senha, zero dado pessoal.
-- R6: gravados na MESMA transação da tabela de origem (users/sessions).
--     Dono da escrita: internal/repository/auth_repo.go
-- R8: auth_lookup é a FONTE DE VERDADE da unicidade de e-mail.
CREATE TABLE auth_lookup (
    email     CITEXT PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id   UUID NOT NULL
);

CREATE TABLE session_lookup (
    token_hash VARCHAR(255) PRIMARY KEY,
    tenant_id  UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id    UUID NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX ix_session_lookup_expires ON session_lookup(expires_at);

-- Fluxo obrigatório:
-- 1. Request chega sem tenant conhecido.
-- 2. Middleware lê session_lookup (cookie) ou auth_lookup (POST /login).
-- 3. Abre Tx e emite SET LOCAL app.tenant_id = <encontrado>.
--    SET LOCAL, nunca SET — o pool pgx reusa conexão.
-- 4. Toda operação seguinte roda sob FORCE RLS.

-- ---------- roles ----------
CREATE TABLE roles (
    tenant_id   UUID NOT NULL,
    id          UUID NOT NULL DEFAULT gen_random_uuid(),
    name        VARCHAR(100) NOT NULL,
    description TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, id),
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE RESTRICT,
    UNIQUE (tenant_id, name)
);
CREATE TRIGGER tr_roles_updated BEFORE UPDATE ON roles
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();
ALTER TABLE roles ENABLE ROW LEVEL SECURITY;
ALTER TABLE roles FORCE  ROW LEVEL SECURITY;
CREATE POLICY p_roles ON roles FOR ALL
  USING      (tenant_id = current_setting('app.tenant_id')::UUID)
  WITH CHECK (tenant_id = current_setting('app.tenant_id')::UUID);

-- ---------- role_permissions ----------
CREATE TABLE role_permissions (
    tenant_id UUID NOT NULL,
    role_id   UUID NOT NULL,
    module    VARCHAR(100) NOT NULL,
    action    VARCHAR(50)  NOT NULL,
    PRIMARY KEY (tenant_id, role_id, module, action),
    FOREIGN KEY (tenant_id, role_id) REFERENCES roles(tenant_id, id) ON DELETE RESTRICT
);
ALTER TABLE role_permissions ENABLE ROW LEVEL SECURITY;
ALTER TABLE role_permissions FORCE  ROW LEVEL SECURITY;
CREATE POLICY p_role_permissions ON role_permissions FOR ALL
  USING      (tenant_id = current_setting('app.tenant_id')::UUID)
  WITH CHECK (tenant_id = current_setting('app.tenant_id')::UUID);

-- ---------- users ----------
CREATE TABLE users (
    tenant_id            UUID NOT NULL,
    id                   UUID NOT NULL DEFAULT gen_random_uuid(),
    role_id              UUID NOT NULL,
    email                CITEXT NOT NULL,
    password_hash        VARCHAR(255) NULL,       -- Argon2id
    auth_provider        VARCHAR(20) NOT NULL DEFAULT 'local'
                         CHECK (auth_provider IN ('local','google')),
    provider_sub         VARCHAR(255) UNIQUE NULL,
    must_change_password BOOLEAN NOT NULL DEFAULT TRUE,
    status               VARCHAR(50) NOT NULL DEFAULT 'active',
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_login_at        TIMESTAMPTZ NULL,
    PRIMARY KEY (tenant_id, id),
    FOREIGN KEY (tenant_id)          REFERENCES tenants(id)          ON DELETE RESTRICT,
    FOREIGN KEY (tenant_id, role_id) REFERENCES roles(tenant_id, id) ON DELETE RESTRICT,
    UNIQUE (tenant_id, email),
    CONSTRAINT chk_auth_provider CHECK (
        (auth_provider = 'local'  AND password_hash IS NOT NULL) OR
        (auth_provider = 'google' AND provider_sub  IS NOT NULL)
    )
);
CREATE TRIGGER tr_users_updated BEFORE UPDATE ON users
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE users FORCE  ROW LEVEL SECURITY;
CREATE POLICY p_users ON users FOR ALL
  USING      (tenant_id = current_setting('app.tenant_id')::UUID)
  WITH CHECK (tenant_id = current_setting('app.tenant_id')::UUID);
-- R8: unicidade global de e-mail é garantida por auth_lookup.email (PK).

-- ---------- user_scopes  (C1 CORRIGIDO) ----------
CREATE TABLE user_scopes (
    tenant_id  UUID NOT NULL,
    id         UUID NOT NULL DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL,
    scope_type VARCHAR(50) NOT NULL CHECK (scope_type IN ('all','company','branch')),
    scope_id   UUID NULL,
    PRIMARY KEY (tenant_id, id),
    FOREIGN KEY (tenant_id, user_id) REFERENCES users(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT chk_scope_id CHECK (
        (scope_type =  'all' AND scope_id IS NULL) OR
        (scope_type <> 'all' AND scope_id IS NOT NULL)
    )
);
CREATE UNIQUE INDEX ux_user_scopes ON user_scopes(
    tenant_id, user_id, scope_type,
    COALESCE(scope_id, '00000000-0000-0000-0000-000000000000'::UUID)
);
ALTER TABLE user_scopes ENABLE ROW LEVEL SECURITY;
ALTER TABLE user_scopes FORCE  ROW LEVEL SECURITY;
CREATE POLICY p_user_scopes ON user_scopes FOR ALL
  USING      (tenant_id = current_setting('app.tenant_id')::UUID)
  WITH CHECK (tenant_id = current_setting('app.tenant_id')::UUID);

-- ---------- sessions ----------
CREATE TABLE sessions (
    tenant_id  UUID NOT NULL,
    id         UUID NOT NULL DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL,
    token_hash VARCHAR(255) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked    BOOLEAN NOT NULL DEFAULT FALSE,
    ip_address VARCHAR(45) NULL,
    user_agent TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, id),
    FOREIGN KEY (tenant_id, user_id) REFERENCES users(tenant_id, id) ON DELETE CASCADE
);
CREATE INDEX ix_sessions_tenant_user ON sessions(tenant_id, user_id);
ALTER TABLE sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE sessions FORCE  ROW LEVEL SECURITY;
CREATE POLICY p_sessions ON sessions FOR ALL
  USING      (tenant_id = current_setting('app.tenant_id')::UUID)
  WITH CHECK (tenant_id = current_setting('app.tenant_id')::UUID);

-- ---------- company ----------
CREATE TABLE company (
    tenant_id  UUID NOT NULL,
    id         UUID NOT NULL DEFAULT gen_random_uuid(),
    nome       VARCHAR(255) NOT NULL,
    cnpj       VARCHAR(14)  NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, id),
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE RESTRICT,
    UNIQUE (tenant_id, cnpj)
);
CREATE TRIGGER tr_company_updated BEFORE UPDATE ON company
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();
ALTER TABLE company ENABLE ROW LEVEL SECURITY;
ALTER TABLE company FORCE  ROW LEVEL SECURITY;
CREATE POLICY p_company ON company FOR ALL
  USING      (tenant_id = current_setting('app.tenant_id')::UUID)
  WITH CHECK (tenant_id = current_setting('app.tenant_id')::UUID);

-- ---------- branch ----------
CREATE TABLE branch (
    tenant_id  UUID NOT NULL,
    id         UUID NOT NULL DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL,
    nome       VARCHAR(255) NOT NULL,
    cnpj       VARCHAR(14)  NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, id),
    FOREIGN KEY (tenant_id, company_id) REFERENCES company(tenant_id, id) ON DELETE RESTRICT,
    UNIQUE (tenant_id, cnpj)
);
CREATE INDEX ix_branch_tenant_company ON branch(tenant_id, company_id);
CREATE TRIGGER tr_branch_updated BEFORE UPDATE ON branch
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();
ALTER TABLE branch ENABLE ROW LEVEL SECURITY;
ALTER TABLE branch FORCE  ROW LEVEL SECURITY;
CREATE POLICY p_branch ON branch FOR ALL
  USING      (tenant_id = current_setting('app.tenant_id')::UUID)
  WITH CHECK (tenant_id = current_setting('app.tenant_id')::UUID);

-- ---------- sector  (R2: chave alternativa p/ FK composta) ----------
CREATE TABLE sector (
    tenant_id  UUID NOT NULL,
    id         UUID NOT NULL DEFAULT gen_random_uuid(),
    branch_id  UUID NOT NULL,
    nome       VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, id),
    FOREIGN KEY (tenant_id, branch_id) REFERENCES branch(tenant_id, id) ON DELETE RESTRICT,
    UNIQUE (tenant_id, branch_id, id)   -- R2: permite FK (tenant,branch,sector)
);
CREATE INDEX ix_sector_tenant_branch ON sector(tenant_id, branch_id);
CREATE TRIGGER tr_sector_updated BEFORE UPDATE ON sector
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();
ALTER TABLE sector ENABLE ROW LEVEL SECURITY;
ALTER TABLE sector FORCE  ROW LEVEL SECURITY;
CREATE POLICY p_sector ON sector FOR ALL
  USING      (tenant_id = current_setting('app.tenant_id')::UUID)
  WITH CHECK (tenant_id = current_setting('app.tenant_id')::UUID);

-- ---------- collaborator  (R2 aplicado) ----------
CREATE TABLE collaborator (
    tenant_id  UUID NOT NULL,
    id         UUID NOT NULL DEFAULT gen_random_uuid(),
    branch_id  UUID NOT NULL,
    sector_id  UUID NOT NULL,
    nome       VARCHAR(255) NOT NULL,
    document   VARCHAR(20)  NOT NULL,   -- dado pessoal · retenção LGPD
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, id),
    FOREIGN KEY (tenant_id, branch_id) REFERENCES branch(tenant_id, id) ON DELETE RESTRICT,
    -- R2: setor obrigatoriamente da MESMA filial
    FOREIGN KEY (tenant_id, branch_id, sector_id)
        REFERENCES sector(tenant_id, branch_id, id) ON DELETE RESTRICT
);
CREATE INDEX ix_collaborator_tenant_branch ON collaborator(tenant_id, branch_id);
CREATE TRIGGER tr_collaborator_updated BEFORE UPDATE ON collaborator
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();
ALTER TABLE collaborator ENABLE ROW LEVEL SECURITY;
ALTER TABLE collaborator FORCE  ROW LEVEL SECURITY;
CREATE POLICY p_collaborator ON collaborator FOR ALL
  USING      (tenant_id = current_setting('app.tenant_id')::UUID)
  WITH CHECK (tenant_id = current_setting('app.tenant_id')::UUID);

-- ---------- asset_type ----------
CREATE TABLE asset_type (
    tenant_id  UUID NOT NULL,
    id         UUID NOT NULL DEFAULT gen_random_uuid(),
    nome       VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, id),
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE RESTRICT,
    UNIQUE (tenant_id, nome)
);
CREATE TRIGGER tr_asset_type_updated BEFORE UPDATE ON asset_type
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();
ALTER TABLE asset_type ENABLE ROW LEVEL SECURITY;
ALTER TABLE asset_type FORCE  ROW LEVEL SECURITY;
CREATE POLICY p_asset_type ON asset_type FOR ALL
  USING      (tenant_id = current_setting('app.tenant_id')::UUID)
  WITH CHECK (tenant_id = current_setting('app.tenant_id')::UUID);

-- ---------- asset  (R2 aplicado) ----------
CREATE TABLE asset (
    tenant_id     UUID NOT NULL,
    id            UUID NOT NULL DEFAULT gen_random_uuid(),
    asset_type_id UUID NOT NULL,
    branch_id     UUID NOT NULL,
    sector_id     UUID NOT NULL,
    nome          VARCHAR(255) NOT NULL,
    serial_number VARCHAR(100) NULL,
    status        VARCHAR(50) NOT NULL DEFAULT 'available',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, id),
    FOREIGN KEY (tenant_id, asset_type_id)
        REFERENCES asset_type(tenant_id, id) ON DELETE RESTRICT,
    FOREIGN KEY (tenant_id, branch_id)
        REFERENCES branch(tenant_id, id) ON DELETE RESTRICT,
    -- R2: setor obrigatoriamente da MESMA filial
    FOREIGN KEY (tenant_id, branch_id, sector_id)
        REFERENCES sector(tenant_id, branch_id, id) ON DELETE RESTRICT
);
CREATE INDEX ix_asset_tenant_type   ON asset(tenant_id, asset_type_id);
CREATE INDEX ix_asset_tenant_branch ON asset(tenant_id, branch_id);
CREATE UNIQUE INDEX ux_asset_serial ON asset(tenant_id, serial_number)
    WHERE serial_number IS NOT NULL;
CREATE TRIGGER tr_asset_updated BEFORE UPDATE ON asset
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();
ALTER TABLE asset ENABLE ROW LEVEL SECURITY;
ALTER TABLE asset FORCE  ROW LEVEL SECURITY;
CREATE POLICY p_asset ON asset FOR ALL
  USING      (tenant_id = current_setting('app.tenant_id')::UUID)
  WITH CHECK (tenant_id = current_setting('app.tenant_id')::UUID);

-- ---------- allocation ----------
CREATE TABLE allocation (
    tenant_id              UUID NOT NULL,
    id                     UUID NOT NULL DEFAULT gen_random_uuid(),
    asset_id               UUID NOT NULL,
    collaborator_id        UUID NOT NULL,
    allocated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    returned_at            TIMESTAMPTZ NULL,
    termo_responsabilidade TEXT NULL,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, id),
    FOREIGN KEY (tenant_id, asset_id)
        REFERENCES asset(tenant_id, id) ON DELETE RESTRICT,
    FOREIGN KEY (tenant_id, collaborator_id)
        REFERENCES collaborator(tenant_id, id) ON DELETE RESTRICT
);
-- 1 alocação ativa por ativo
CREATE UNIQUE INDEX ux_allocation_ativa ON allocation(tenant_id, asset_id)
    WHERE returned_at IS NULL;
CREATE INDEX ix_allocation_tenant_collab ON allocation(tenant_id, collaborator_id);
CREATE TRIGGER tr_allocation_updated BEFORE UPDATE ON allocation
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();
ALTER TABLE allocation ENABLE ROW LEVEL SECURITY;
ALTER TABLE allocation FORCE  ROW LEVEL SECURITY;
CREATE POLICY p_allocation ON allocation FOR ALL
  USING      (tenant_id = current_setting('app.tenant_id')::UUID)
  WITH CHECK (tenant_id = current_setting('app.tenant_id')::UUID);
-- Aceito conscientemente: coerência asset.branch = collaborator.branch é
-- validada na camada de serviço, não no banco.

-- ---------- audit_log  (C4: append-only) ----------
CREATE TABLE audit_log (
    tenant_id   UUID NOT NULL,
    id          UUID NOT NULL DEFAULT gen_random_uuid(),
    actor_id    UUID NULL,
    action      VARCHAR(100) NOT NULL,
    entity_name VARCHAR(100) NOT NULL,
    entity_id   UUID NOT NULL,
    old_data    JSONB NULL,
    new_data    JSONB NULL,
    ip_address  VARCHAR(45) NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, id),
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE RESTRICT
);
CREATE INDEX ix_audit_log_tenant_entity ON audit_log(tenant_id, entity_name, entity_id);
CREATE INDEX ix_audit_log_created       ON audit_log(tenant_id, created_at);
ALTER TABLE audit_log ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_log FORCE  ROW LEVEL SECURITY;
-- C4: políticas separadas — NUNCA FOR ALL.
CREATE POLICY p_audit_log_select ON audit_log FOR SELECT
  USING (tenant_id = current_setting('app.tenant_id')::UUID);
CREATE POLICY p_audit_log_insert ON audit_log FOR INSERT
  WITH CHECK (tenant_id = current_setting('app.tenant_id')::UUID);
REVOKE UPDATE, DELETE ON audit_log FROM invtech_app;

-- ---------- import_batches ----------
CREATE TABLE import_batches (
    tenant_id         UUID NOT NULL,
    id                UUID NOT NULL DEFAULT gen_random_uuid(),
    idempotency_key   VARCHAR(255) NOT NULL,
    status            VARCHAR(50) NOT NULL DEFAULT 'processing',
    records_processed INTEGER NOT NULL DEFAULT 0,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, id),
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE RESTRICT,
    UNIQUE (tenant_id, idempotency_key)
);
ALTER TABLE import_batches ENABLE ROW LEVEL SECURITY;
ALTER TABLE import_batches FORCE  ROW LEVEL SECURITY;
CREATE POLICY p_import_batches ON import_batches FOR ALL
  USING      (tenant_id = current_setting('app.tenant_id')::UUID)
  WITH CHECK (tenant_id = current_setting('app.tenant_id')::UUID);

-- ---------- Grants finais ----------
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO invtech_app;
-- Reafirmar C4 após o GRANT amplo:
REVOKE UPDATE, DELETE ON audit_log, errors_log FROM invtech_app;
