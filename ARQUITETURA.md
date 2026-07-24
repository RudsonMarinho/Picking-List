# ARQUITETURA.md — InvTech SaaS (Fase 1)
> Dev Factory Elite · Etapa 1 revisada · Etapa 2 APROVADA COM CONDIÇÕES
> requestId: dfe-invtech-saas-etapa0-2026-07-19
> Emendas do Mentor C1..C6 + FAIL_FAST + R1..R10 APLICADAS neste documento.
> Amendment 6 (SQLC-01) APLICADA — ver nota abaixo.

## Emenda SQLC-01 (amendment 6, aprovada pelo Mentor em 2026-07-23)

**Original:** stack previa sqlc para geração de queries tipadas.
**Alterado (só nesta Fase 1 do invtech — não altera o Split v3.1 nem cria precedente):**
queries pgx v5 parametrizadas escritas à mão em `internal/repository/*.go`.

**Justificativa:** decisão pragmática do executor durante o W4, por tempo de execução —
não houve pausa para consulta prévia ao Mentor antes de implementar o desvio (processo
incorreto, registrado como lição — ver PROGRESSO.md). O Mentor aceitou o mérito da
decisão e formalizou a emenda com condições:
1. registrar aqui a decisão com justificativa (este bloco);
2. todo caminho de query coberto por teste de integração contra Postgres real — feito:
   `internal/app/business_flow_test.go` + `internal/app/module_crud_coverage_test.go`
   exercitam create/list/update/delete dos 9 módulos + import + allocation/return via
   API real contra Postgres real (build tag `integration`);
3. `queries/` removida (estava vazia — nunca chegou a ser usada).

**Gatilhos de readoção do sqlc:** primeiro bug de descasamento query↔struct em produção,
ou início da Fase 2 (billing exige tipagem verificada).

## Decisões

| Decisão | Valor |
|---|---|
| Stack | Go 1.23 + Fiber v2 + pgx v5 (queries à mão, ver SQLC-01) + golang-migrate |
| Banco | PostgreSQL isolado — invtech_db |
| Servidor | srv110 (10.0.0.110) · Coolify grupo `servicos-saas` |
| Gateway Docker | 10.0.0.110 |
| Repo | Fortcargo/invtech |
| Frontend Fase 1 | //go:embed do HTML atual (sem Vercel) |
| Versões Docker | sempre pinadas — nunca :latest |
| Error ID Prefix | INV |
| Senhas | Argon2id |
| Sessão | server-side revogável (tabela sessions) |

## Diagrama de serviços

Navegador → pfSense/HAProxy (SNAT) → Traefik :443
  → invtech-api :8080 (Go · distroless · UI via //go:embed)
      → PostgreSQL invtech_db (RLS FORCE)

Comunicação interna pela rede Docker do Coolify.
A API não é publicada no host (expose, não ports).

## Estrutura de pastas

/
├── .claude/CLAUDE.md
├── AGENTS.md · GEMINI.md · SESSAO.md · HANDOFF.md · PROGRESSO.md
├── ARQUITETURA.md · README.md
├── cmd/server/main.go
├── internal/{app,handlers,services,repository,middleware,seed}/
├── pkg/{config,logger,security}/
├── migrations/
│   ├── 000000_create_role.{up,down}.sql
│   ├── 000001_create_errors_log.{up,down}.sql
│   ├── 000002_create_tenants.{up,down}.sql
│   └── 000003_create_business_schema.{up,down}.sql
│   └── embed.go            # //go:embed *.sql — golang-migrate roda embutido (-migrate)
├── web/index.html          # HTML atual preservado, alvo do //go:embed
├── scripts/bootstrap_db.sh # C3 — GRANT CONNECT fora das migrations
├── Dockerfile · docker-compose.yaml · check-ports.sh · .env.example
└── go.mod · go.sum

## Schema do banco

```sql
-- ============================================================
-- 000000_create_role.up.sql
-- Executado pelo OWNER (DB_MIGRATE_USER). invtech_app é runtime.
-- C3: GRANT CONNECT NÃO fica aqui — golang-migrate não interpola
--     variáveis. Vai em scripts/bootstrap_db.sh.
-- ============================================================
CREATE EXTENSION IF NOT EXISTS "citext";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE ROLE invtech_app WITH LOGIN
  NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS;
-- Senha definida no provisionamento (Coolify), nunca em migration.

GRANT USAGE ON SCHEMA public TO invtech_app;

-- C2: alcança TODAS as tabelas futuras, inclusive 000004+ (Fase 2).
ALTER DEFAULT PRIVILEGES IN SCHEMA public
  GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO invtech_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
  GRANT USAGE, SELECT ON SEQUENCES TO invtech_app;

-- ============================================================
-- 000001_create_errors_log.up.sql
-- ============================================================
CREATE TABLE errors_log (
    error_id        VARCHAR(12) PRIMARY KEY,
    request_id      UUID NOT NULL,
    tenant_id       UUID NULL,
    occurred_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    endpoint        VARCHAR(255),
    http_method     VARCHAR(10),
    http_status     INTEGER,
    exception_type  VARCHAR(255),
    exception_msg   TEXT,
    traceback       TEXT,
    request_payload JSONB,   -- mascarar password/token/secret ANTES do INSERT
    user_context    VARCHAR(255),
    environment     VARCHAR(20) DEFAULT 'production'
);
CREATE INDEX ix_errors_log_occurred ON errors_log(occurred_at);

-- C4: append-only para o runtime.
REVOKE UPDATE, DELETE ON errors_log FROM invtech_app;
-- Retenção 90 dias: expurgo com role de manutenção, nunca invtech_app.

-- ============================================================
-- 000002_create_tenants.up.sql
-- ============================================================
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

-- ============================================================
-- 000003_create_business_schema.up.sql
-- ============================================================
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

-- ============================================================
-- 000004_extend_asset_collaborator_fields.up.sql
-- Campos que a UI original (web/index.html) já exibia e o schema da Fase 1
-- não tinha — adicionados por decisão do usuário ao conectar a UI ao
-- backend real, em vez de simplificar a UI pro schema mínimo.
-- ============================================================
ALTER TABLE asset
    ADD COLUMN hostname VARCHAR(255) NULL,
    ADD COLUMN brand     VARCHAR(100) NULL,
    ADD COLUMN model     VARCHAR(100) NULL,
    ADD COLUMN cpu       VARCHAR(100) NULL,
    ADD COLUMN ram       VARCHAR(50)  NULL,
    ADD COLUMN storage   VARCHAR(100) NULL,
    ADD COLUMN notes     TEXT         NULL;

ALTER TABLE collaborator
    ADD COLUMN cargo VARCHAR(150) NULL;
```

### Extensões de schema 000005/000006 — casar com a UI original

Ao conectar `web/index.html` (nunca antes ligada ao backend) à API real, mais
campos que a UI já exibia não existiam no schema mínimo da Fase 1. Decisão do
usuário: estender o backend em vez de simplificar a UI (mesmo espírito da
000004). Resumo (SQL completo nos arquivos de migration):

| Migration | Tabela | Campos adicionados |
|---|---|---|
| 000005 | `asset` | `asset_tag` (código de patrimônio digitado, distinto do `id` UUID e do `serial_number`), único por tenant quando preenchido |
| 000006 | `company` | `city`, `state` |
| 000006 | `branch` | `city`, `state` |
| 000006 | `sector` | `description` |
| 000006 | `asset_type` | `icon`, `description` |
| 000006 | `collaborator` | `email`, `status` (default `active`); `document` (CPF) virou opcional — a UI original não coleta |

`asset.status` não tem CHECK constraint — além de `available`/`allocated`
(controlados pelo fluxo de alocação/devolução), a UI permite alternar
manualmente para `maintenance` via edição (`UpdateAsset` aceita `status`
opcional, só altera se enviado — `COALESCE` no UPDATE).

## Segurança da aplicação

### FAIL-FAST no boot (BLOQUEADOR — implementar antes de tudo)
Se `APP_ENV=production` e qualquer uma for verdadeira, registrar erro fatal
e ABORTAR o processo (não subir degradado):
- `MAIL_DRIVER != provider`
- `AUTO_CONFIRM_SIGNUP = true`
- seed habilitado
- rota `/api/v1/dev/*` registrada
Rotas `/dev/*` idealmente não compiladas em produção (build tag).
Teste automatizado obrigatório provando o abort.

### C6 — CSRF
Todas as rotas de mutação (POST/PUT/PATCH/DELETE) exigem token CSRF por
double-submit: cookie `csrf_token` (não-HttpOnly, `SameSite=Lax`) + header
`X-CSRF-Token`, comparados em constant-time. Sem token, ou token divergente,
retorna 403 antes de qualquer acesso ao banco.

### R1 — Domínios
Vercel (Fase 2) fica em subdomínio do MESMO domínio raiz do app
(`www.<raiz>` e `app.<raiz>`) para manter cookie `SameSite=Lax`.
`SameSite=None` está descartado.

### Sessão e senha
- Argon2id (parâmetros documentados no README).
- Cookie de sessão: `HttpOnly; Secure; SameSite=Lax`.
- Sessão server-side revogável; logout invalida a linha em `sessions` e
  remove de `session_lookup` na mesma transação.
- `must_change_password=true` força troca antes de qualquer rota de negócio.

### Nunca filtrar por IP de origem
O pfSense faz SNAT — toda origem externa chega como `10.0.0.254`.

### RBAC — enforcement de role_permissions/user_scopes (implementado 2026-07-23)
`role_permissions(role_id, module, action)` e `user_scopes(user_id, scope_type, scope_id)` são
gravados desde o W4, mas só passaram a ser lidos nesta data — antes, qualquer usuário autenticado
acessava qualquer módulo sem respeitar role/escopo.
- **Permissão**: `middleware.RequirePermission(module, action)` em toda rota de negócio
  (`internal/handlers/routes.go`), exceto `/dashboard` (usada pelo frontend só pra checar se a
  sessão está viva). 403 se a role não tiver o par exato.
- **Escopo**: aplicado em create/update/delete de companies/branches/sectors/collaborators/
  assets/allocations via `middleware.Scope(c)` (`internal/handlers/authz.go`). Escopo `'company'`
  expande pra todas as filiais da empresa; `'branch'` fica só na filial (não sobe pra empresa).
  Criar empresa exige escopo `'all'`. `asset_types`/`users`/`roles`/`import` só têm gate de
  permissão, não de escopo (catálogo tenant-wide / administração do tenant / payload heterogêneo).
- **Leitura (list/get) não é filtrada por escopo** — decisão consciente, ver SESSAO.md pra detalhe
  e trade-off. Pendência real se precisar de granularidade também na leitura.

## Contrato da API

| Método | Endpoint | Entrada | Resposta |
|---|---|---|---|
| POST | /api/v1/auth/login | { email, password } | Cookie HttpOnly + { user_info } |
| POST | /api/v1/auth/logout | — | { status } |
| POST | /api/v1/auth/change-password | { old, new } | { status } |
| GET  | /api/v1/dashboard | — | { stats } |
| GET/POST | /api/v1/companies | { nome, cnpj } | company |
| PUT/DELETE | /api/v1/companies/:id | { nome, cnpj } | company |
| GET/POST | /api/v1/companies/:id/branches | { nome, cnpj } | branch |
| PUT/DELETE | /api/v1/branches/:id | { nome, cnpj } | branch |
| GET/POST | /api/v1/branches/:id/sectors | { nome } | sector |
| PUT/DELETE | /api/v1/sectors/:id | { nome } | sector |
| GET/POST | /api/v1/collaborators | { branch_id, sector_id, nome, document } | collaborator |
| PUT/DELETE | /api/v1/collaborators/:id | idem | collaborator |
| GET/POST | /api/v1/asset-types | { nome } | asset_type |
| PUT/DELETE | /api/v1/asset-types/:id | { nome } | asset_type |
| GET/POST | /api/v1/assets | { asset_type_id, branch_id, sector_id, nome, serial_number } | asset |
| PUT/DELETE | /api/v1/assets/:id | idem | asset |
| GET/POST | /api/v1/allocations | { asset_id, collaborator_id } | { allocation, termo_html } |
| POST | /api/v1/allocations/:id/return | — | { status } |
| GET/POST | /api/v1/users | { email, role_id, senha_temporaria, scopes[] } | user |
| PUT/DELETE | /api/v1/users/:id | idem | user |
| GET/POST | /api/v1/roles | { name, permissions[] } | role |
| POST | /api/v1/import | { idempotency_key, data[] } | { status, count } |
| GET  | /health | — | { status: "ok" } |
| GET  | /health/ready | — | { db, memory } |

DELETE respeita ON DELETE RESTRICT: tipo/setor/filial em uso retorna 409.

## Variáveis de ambiente

```env
APP_PORT=${APP_PORT}
APP_ENV=${APP_ENV}
ERROR_ID_PREFIX=${ERROR_ID_PREFIX}

DB_HOST=${DB_HOST}
DB_PORT=${DB_PORT}
DB_NAME=${DB_NAME}
DB_USER=${DB_USER}                       # invtech_app — runtime
DB_PASSWORD=${DB_PASSWORD}
DB_MIGRATE_USER=${DB_MIGRATE_USER}       # owner — só migrations
DB_MIGRATE_PASSWORD=${DB_MIGRATE_PASSWORD}

SESSION_COOKIE_NAME=${SESSION_COOKIE_NAME}
CSRF_COOKIE_NAME=${CSRF_COOKIE_NAME}

MAIL_DRIVER=${MAIL_DRIVER}               # log | noop | provider
AUTO_CONFIRM_SIGNUP=${AUTO_CONFIRM_SIGNUP}
FEATURE_PUBLIC_SIGNUP=${FEATURE_PUBLIC_SIGNUP}
FEATURE_BILLING=${FEATURE_BILLING}
FEATURE_EMAIL_DELIVERY=${FEATURE_EMAIL_DELIVERY}
```
Proibido `DATABASE_URL` / URI com `@` — conexão via `pgx.ParseConfig()`.
Proibido commitar placeholder descritivo no `.env.example` (o Coolify puxa
como valor real) — usar string vazia ou literal genérico.

## docker-compose.yaml

> Nome do arquivo é `docker-compose.yaml` (não `.yml`) — é o path default que o
> build pack `dockercompose` do Coolify procura (`/docker-compose.yaml`); usar
> `.yml` faz o Coolify não encontrar o arquivo e cair num fallback silencioso
> (imagem `nginx:alpine` estática, porta 80). Docker Compose CLI reconhece os
> dois nomes, então isso não muda nada no `docker compose up -d` local.

```yaml
services:
  invtech-api:
    build:
      context: .
    image: invtech-api:1.0.0
    restart: unless-stopped
    expose:
      - "8080"                 # R3 — Traefik alcança pela rede Docker
    mem_limit: 512m
    pids_limit: 100
    env_file:
      - .env
    healthcheck:
      test: ["CMD", "/app/invtech-api", "-healthcheck"]   # R4 — path absoluto
      interval: 30s
      timeout: 5s
      retries: 3
      start_period: 15s
    networks:
      - default
      - coolify               # invtech-db vive nela — sem isso, DB_HOST não resolve (achado no
                               # primeiro deploy real: /health 200, /health/ready 503)

networks:
  coolify:
    external: true
```
Dockerfile multi-stage `golang:1.23-alpine` → `gcr.io/distroless/static-debian12`.
Binário em `/app/invtech-api`. Healthcheck do Coolify na rota `/`.

**Rede `coolify`:** o build pack `dockercompose` do Coolify isola cada app numa rede docker
própria (nome = uuid do recurso). `invtech-db` roda solto na rede `coolify` (não num
docker-compose gerenciado por este build pack), então sem declarar essa rede aqui o
`invtech-api` não a enxerga — `DB_HOST` fica irresolúvel e `/health/ready` fica 503 mesmo com
`/health` respondendo 200. Confirmado que essa declaração é durável (sobrevive a redeploys),
diferente do que `CLAUDE.md` documenta para outros apps deste host.

**Incidente 2026-07-23 (Gateway Timeout em produção):** a primeira tentativa desse fix listava
`networks: [default, coolify]` no serviço. Como não há `networks.default` declarada no topo do
arquivo, o Docker Compose criou uma **terceira** rede ad-hoc (`<uuid-do-app>_default`) pra
satisfazer a referência não declarada — rede que o `coolify-proxy` não está conectado. O Traefik
às vezes escolhia essa rede como destino do backend, e a requisição pendurava até timeout (site
inteiro fora do ar, `/health` respondendo normal só internamente). Corrigido removendo `default`
da lista — só `coolify` fica explícito; o Coolify já injeta sozinho a rede isolada própria do
app. **Nunca adicionar uma rede chamada `default` a um serviço sem declarar
`networks: { default: ... }` no topo do arquivo** — o Compose cria uma nova silenciosamente.

## Suíte negativa (VERDE antes de qualquer handler)

- query como tenant B retorna 0 linhas do tenant A (todas as tabelas)
- INSERT com tenant_id alheio é rejeitado (WITH CHECK)
- `invtech_app` não consegue UPDATE nem DELETE em `audit_log` e `errors_log`
- `scope_type='all'` com `scope_id NULL` insere com sucesso
- mutação sem token CSRF é rejeitada com 403
- `asset` não aceita `sector_id` de `branch` diferente do seu `branch_id`
- `collaborator` idem
- 2ª alocação ativa do mesmo `asset` é rejeitada
- `MAIL_DRIVER=noop`: criar tenant + usuário + login funcionam
- `APP_ENV=production` com bypass ativo → processo aborta no boot
- `must_change_password=true` bloqueia rotas de negócio até a troca
- request sem `SET LOCAL app.tenant_id` falha explicitamente (fail-closed)

## Como rodar e testar sem dependências externas

1. `docker compose up -d`
2. `docker compose exec invtech-api /app/invtech-api -seed`
3. Login com o admin do tenant demo criado pelo seed
4. Troca de senha forçada (`must_change_password=true`)
5. Navegar os 9 módulos com dados de exemplo — sem e-mail, sem provedor
   de pagamento, sem internet

Seed cria (R10): plano `trial` → tenant demo → role admin + permissões →
usuário admin → empresa → filial → setor → colaborador → tipo de ativo →
ativo → alocação com Termo de Responsabilidade gerado.

## Ordem de implementação (Etapa 4)

1. FAIL-FAST de boot — antes de qualquer migration
2. Migrations 000000..000003
3. Suíte negativa VERDE
4. //go:embed da UI + Argon2id + sessions + lookups + CSRF
5. Handlers dos 9 módulos + import idempotente
6. Seed

## Fase 2

Ver `ARQUITETURA_FASE2.md` (signup, trial 30d, billing, backoffice /admin,
Vercel). Migrations 000004+. Exige nova MentorDecision antes de iniciar.
Emenda C5 aplica-se lá: DROP/CREATE POLICY das 15 tabelas da Fase 1
acrescentando `OR current_setting('app.is_platform_admin', true) = 'on'`,
escrito por extenso. PROIBIDO BYPASSRLS.
