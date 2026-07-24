# PROGRESSO.md — InvTech

## Feito
- **W0** ✅ setup-remotes.sh parametrizado (`--org/--server/--dry-run`).
- **W1** ✅ ARQUITETURA.md definida (amendment 4) — na raiz deste repo.
- **W2** ✅ scaffold + infra:
  - árvore cmd/internal/pkg/migrations(vazio)/queries/web/scripts
  - config: go.mod, docker-compose.yaml, Dockerfile, .env.example, check-ports.sh
  - governança/segurança do _template: SELF-HEALER, SEGREDOS, .githooks, ORQUESTRADOR, AGENTS
  - web/index.html (HTML de origem, checksum validado)
  - repo Fortcargo/invtech criado + remotes (origin + servidor) + push
  - Coolify srv110: grupo servicos-saas + Postgres invtech_db (postgres:16-alpine, healthy)
- **W3** ✅ gate-foundry (PILOT-FOUNDRY-01) — decisão NO_AGENT_REQUIRED → sessionFallback.
  Sessão dedicada `claude-remote@invtech` ativa/enabled no srv110 (cwd correto, INV-1=1, idle).
  Detalhe: `docs/W3-gate-decision.md`. **Retornado ao Mentor.**
- **W4** ✅ implementação completa, liberada pela MentorDecision amendment 5 (P1/P2/P3):
  - FAIL-FAST no boot (testado: aborta em produção com MAIL_DRIVER≠provider,
    AUTO_CONFIRM_SIGNUP=true, seed habilitado ou rotas /dev/* registradas).
  - P1 lado código: `scripts/bootstrap_db.sh` (senha + GRANT CONNECT do invtech_app,
    idempotente) + `.env.example` com DB_USER≠DB_MIGRATE_USER documentado.
  - Migrations 000000..000003 aplicadas e conferidas contra Postgres real (RLS FORCE,
    políticas, grants, triggers, extensões citext/pgcrypto).
  - Suíte negativa 11/11 itens verde (RLS isolation, WITH CHECK, append-only audit/errors_log,
    scope 'all', CSRF 403, FK sector≠branch asset/collaborator, dupla alocação ativa,
    MAIL_DRIVER=noop login, FAIL-FAST abort, must_change_password gate, fail-closed).
  - Auth: Argon2id, sessions revogáveis, auth_lookup/session_lookup mesma transação (R6/R8),
    CSRF double-submit, tenant middleware com Tx por requisição, //go:embed da UI.
  - Handlers dos 9 módulos (dashboard, companies, branches, sectors, collaborators,
    asset-types, assets, allocations, users, roles) + import idempotente
    (`import_batches.idempotency_key`) — validados por teste de integração ponta-a-ponta
    pela API real.
  - Seed (R10) idempotente, testado com o binário real.
  - Binário único com modos `-migrate`/`-seed`/`-healthcheck`/serve, testado via Docker
    (sem `go` no host — toolchain via container `golang:1.23-alpine`).
  - **Desvio registrado:** sqlc não usado — queries pgx escritas à mão. Formalizado como
    emenda SQLC-01 pelo Mentor (amendment 6) — ver ARQUITETURA.md § Emenda SQLC-01.
- **[SYNC VISUAL] pós-W4** ✅ (MentorDecision amendment 6) — gates de deploy G1..G5:
  - **G1** ✅ senha real do `invtech_app` gerada e aplicada no `invtech-db` do Coolify
    (uuid `k43x07e67slriuvc6lh4vdr5`, projeto `servicos-saas`) via `scripts/bootstrap_db.sh`.
    Validado com `SELECT current_user` conectando como `invtech_app` — sucesso. Senha salva
    em `~/infra-credenciais.env` (`INVTECH_APP_DB_PASSWORD`), não versionada.
  - **G2** ✅ `\du invtech_app` no banco real confirma atributos vazios (sem Superuser, sem
    Bypass RLS, sem Create DB/Role); `pg_tables.tableowner` confirma que todas as tabelas
    pertencem a `invtech_owner`, não a `invtech_app`.
  - **G3** ⛔ MCP-FORTCARGO autorizado nesta sessão (2026-07-23), mas `opsmonitor_*` é só
    leitura — o registro em si (`ContainerPlaybook`/`HTTP_ENDPOINTS`) mora em
    `~/projetos/opsmonitor`, que vive no **srv100**, fora do escopo desta sessão de projeto.
    Demanda atualizada com dados concretos:
    `ai-workspace/Documentacao/governanca/demandas/opsmonitor-registrar-hermes-invtech.md`.
  - **G4** ✅ confirmado via banco interno do Coolify: `standalone_postgresqls.image =
    postgres:16-alpine` (uuid `k43x07e67slriuvc6lh4vdr5`).
  - **G5** ✅ checklist de paridade abaixo.
  - Migrations 000000..000003 também já aplicadas no `invtech-db` real (schema criado,
    RLS FORCE confirmada em `roles`/`users`/`asset`/`allocation`/`audit_log`).
- **App `invtech-api` criada no Coolify** (2026-07-23) — projeto `servicos-saas`, environment
  `production`, uuid `j6yc0qzhslky713ra5qte0u4`, build pack `dockercompose`, fonte GitHub via
  GitHub App `fortcargo-github` (repo privado, branch `main`). 17 env vars definidas (config real,
  incluindo credenciais do banco).
- **Domínio `invtech.fortcargo.com.br`** (2026-07-23) — DNS `A → 128.201.243.151` (padrão da
  zona, igual hermes/workflows/automation-engine) + `docker_compose_domains` vinculado ao
  serviço `invtech-api` no Coolify (via PATCH direto na API — a tool MCP
  `coolify110_configurar_app` tem um bug real de schema para esse campo em apps
  `dockercompose`, reportar para quem mantém o `fortcargo-mcp`). Exposição direta aprovada pela
  skill `exposicao-segura` (app tem login próprio, não é painel de infra).
  **Deploy ainda não disparado.**
- **Topologia de git corrigida e validada contra os demais projetos** (2026-07-23) — nenhum
  projeto com working copy em `~/projetos` do srv110 (automation-engine, ai-workspace,
  fortcargo-backend, erp-fortcargo, hermes) tem remote `servidor`; só `origin` (GitHub). O
  srv100 é quem concentra os bare mirrors (`~/repos/*.git`, dezenas de projetos). Invtech
  ajustado para o mesmo padrão: clone no 110 só com `origin`; bare mirror criado e sincronizado
  em `10.0.0.100:~/repos/invtech.git`; bare órfão que eu tinha deixado em `10.0.0.110:~/repos/`
  removido. Mirror em 100 agora sincroniza sozinho: `origin` configurado com refspec de espelho +
  `~/scripts/mirror-invtech.sh` (`git fetch --prune`, com `flock`) + cron `*/30 * * * *` em 100 —
  puramente pull-based, sem depender de push manual do 110.
- **Deploy agora usa GitHub Actions** (2026-07-23) — `.github/workflows/ci-deploy.yml`: job
  `test` (build/vet/unit + Postgres service container + migrations + bootstrap_db.sh + suíte
  negativa/integração completa) → job `deploy` (dispara `GET /api/v1/deploy` no Coolify via
  `https://coolify-fc.fortcargo.com.br`, só se os testes passarem). Secret `COOLIFY_API_TOKEN`
  criado no repo. Correção: o Coolify deste host **tem** endpoint público
  (`coolify-fc.fortcargo.com.br`, rota gerada pelo próprio Coolify) — eu tinha concluído errado
  antes que não tinha. **Achado de segurança fora do escopo:** esse domínio expõe o dashboard
  inteiro do Coolify sem Basic Auth/Cloudflare Access — condição pré-existente, não corrigida
  aqui (decisão maior, arriscava quebrar o próprio webhook).
- **PRIMEIRO DEPLOY REAL** ✅ (2026-07-23) — push → CI → deploy funcionou de ponta a ponta.
  Achado no caminho: `/health` 200 mas `/health/ready` 503 (`pool.Ping` falhando) — causa: build
  pack `dockercompose` isola cada app numa rede docker própria, `invtech-api` não enxergava
  `invtech-db` (rede `coolify`). Fix durável em `docker-compose.yaml` (serviço ganhou
  `networks: [default, coolify]`, `coolify` declarada `external: true`) — testado com um segundo
  deploy real: a rede já veio conectada automaticamente, sem passo manual. Validado ao vivo:
  `https://invtech.fortcargo.com.br/health` e `/health/ready` → 200, `/` (UI) → 200.

### G5 — checklist de paridade dos 9 módulos (contrato da API × implementado)

| Módulo | Endpoints do contrato | Implementado | Coberto por teste de integração |
|---|---|---|---|
| Auth | login, logout, change-password | ✅ | ✅ (login_flow_test.go, business_flow_test.go) |
| Dashboard | GET /dashboard | ✅ | ✅ |
| Companies | GET/POST, PUT/DELETE /:id | ✅ | ✅ (create/update/list/delete) |
| Branches | GET/POST sob company, PUT/DELETE /:id | ✅ | ✅ (create/update/list/delete) |
| Sectors | GET/POST sob branch, PUT/DELETE /:id | ✅ | ✅ (create/update/list/delete) |
| Collaborators | GET/POST, PUT/DELETE /:id | ✅ | ✅ (create/update/list/delete) |
| Asset types | GET/POST, PUT/DELETE /:id | ✅ | ✅ (create/update/list/delete) |
| Assets | GET/POST, PUT/DELETE /:id | ✅ | ✅ (create/update/list/delete) |
| Allocations | GET/POST, POST /:id/return | ✅ | ✅ (create/list/return + 409 dupla ativa) |
| Users | GET/POST, PUT/DELETE /:id | ✅ | ✅ (create via API/list/update/delete) |
| Roles | GET/POST (com permissions) | ✅ | ✅ (create via API com permissions/list) |
| Import | POST /import (idempotente) | ✅ | ✅ (import + reimport idempotente) |

Testes: `internal/app/business_flow_test.go` (fluxo ponta-a-ponta) +
`internal/app/module_crud_coverage_test.go` (update/list/delete de cada módulo +
criação de user/role via API) — ambos rodam contra Postgres real (`-tags=integration`).

## UI conectada ao backend real (2026-07-23) ✅

`web/index.html` era um protótipo 100% client-side (localStorage, login fake, zero `fetch()`),
nunca conectado ao backend construído no W4 — achado só ao criar o primeiro usuário real.
Reescrito por completo:
- Login real + CSRF double-submit + modal de troca de senha obrigatória (novo).
- Os 9 módulos conectados via `/api/v1/*` real (substitui arrays mock + localStorage).
- Alocação virou o modelo histórico real (entidade separada, não campo no ativo).
- Usuários adaptados pro modelo role+scope real.
- 3 migrations novas (000004/000005/000006) estendendo asset/company/branch/sector/asset_type/
  collaborator com os campos que a UI já exibia — decisão do usuário, aplicadas em produção
  antes de cada deploy correspondente.
- Validado ao vivo em produção (curl simulando navegador): CSRF, login, troca de senha,
  CRUD dos 9 módulos com os campos novos, alocação/devolução, role+usuário com escopo — tudo
  contra o tenant real, dados de teste removidos depois.

**Achado de escopo não corrigido:** `role_permissions`/`user_scopes` são gravados mas nenhum
middleware os lê — sem enforcement de RBAC real, qualquer usuário autenticado acessa qualquer
módulo. Registrar como pendência se granularidade de permissão por usuário for necessária.

## Pendente
- **G3:** fica com a sessão `opsmonitor` (srv100) via demanda registrada — fora do escopo desta
  sessão de projeto.
- **RBAC real:** `role_permissions`/`user_scopes` existem mas não são enforced por nenhum
  middleware — avaliar se é necessário pra Fase 2.
- **ai-workspace** (não-bloqueante): registrar PILOT-FOUNDRY-01 em LICOES_APRENDIDAS.md;
  corrigir BASE_DIR do setup_project.py (demanda aberta); remover ARQUITETURA duplicado.
- **Mentor:** Fase 2 exige nova MentorDecision (ARQUITETURA.md § Fase 2).
