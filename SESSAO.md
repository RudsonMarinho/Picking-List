# SESSAO.md — InvTech

> ✅ **W4 formalmente LIBERADO pelo Mentor** — `MENTOR_DECISION_D1_D6_2026-07-23` (`132b005`, decisão D6):
> regras normais do DFE, **sem checkpoint por passo**; voltar ao Mentor só em decisão arquitetural,
> zona cinza da matriz de classe, ou ação Classe C. O W4 já está implementado (estado real abaixo) —
> esta é a confirmação FORMAL. Seguir com amendment 6 e as pendências normais.

## ESTADO ATUAL
**Fase:** UI conectada ao backend real (2026-07-23) — o app está funcionalmente completo e em
produção. Executor dedicado `claude-remote@invtech` **ATIVO** no srv110. MentorDecision
dfe-invtech-saas-etapa0-2026-07-19 amendment 6 em execução.

## UI conectada ao backend real (2026-07-23)

Achado ao criar o primeiro usuário real: `web/index.html` (preservado do W2) nunca tinha sido
conectado ao backend — era um protótipo 100% client-side (localStorage, zero `fetch()`), com
login fake comparando usuário/senha num array JS. Reescrito por completo:

- **Schema estendido em 3 migrations (000004/000005/000006)** pra casar com os campos que a UI
  já exibia e o schema mínimo da Fase 1 não tinha: `asset` ganhou `asset_tag`/`hostname`/`brand`/
  `model`/`cpu`/`ram`/`storage`/`notes`; `company`/`branch` ganharam `city`/`state`; `sector`
  ganhou `description`; `asset_type` ganhou `icon`/`description`; `collaborator` ganhou `email`/
  `status` e `document` virou opcional. Decisão do usuário: estender o backend em vez de
  simplificar a UI. Todas aplicadas em teste e produção, suíte completa verde após cada uma.
- **`web/index.html` inteiramente reescrito**: login real (cookie de sessão + CSRF
  double-submit), modal de troca de senha obrigatória (novo — não existia no protótipo), os 9
  módulos buscando/gravando via `/api/v1/*` real. Alocação virou o modelo histórico real
  (entidade separada, não mais campo no ativo). Usuários adaptados pro modelo role+scope real
  (perfis com módulos = permissão total nesse módulo, escopo all/company/branch).
- **Validado ao vivo em produção** (não só localmente): fluxo completo via curl simulando o
  navegador — CSRF, login, troca de senha (ida e volta, restaurando a senha original do usuário
  real), criação de empresa/filial/setor/tipo/colaborador/ativo com todos os campos novos,
  alocação + devolução, criação de role + usuário com escopo. Dados de teste removidos do tenant
  real depois (via SQL direto — não há endpoint DELETE para allocations/roles, por design).

**Achado de escopo original — CORRIGIDO em 2026-07-23** (ver seção "RBAC: enforcement de
role_permissions/user_scopes implementado" abaixo). `role_permissions`/`user_scopes` eram gravados
mas nenhum middleware os lia; agora são aplicados em todas as rotas de negócio.

**O que existe agora (W4 completo, código + banco real migrado):**
- FAIL-FAST no boot (`pkg/config`), testado (aborta em produção com qualquer bypass ativo).
- P1 resolvido de ponta a ponta: `scripts/bootstrap_db.sh` setou a senha real do
  `invtech_app` no `invtech-db` do Coolify (uuid `k43x07e67slriuvc6lh4vdr5`, projeto
  `servicos-saas`) — **G1/G2 validados com evidência real** (`SELECT current_user` +
  `\du` + `pg_tables.tableowner`). Senha em `~/infra-credenciais.env`
  (`INVTECH_APP_DB_PASSWORD`), não versionada.
- Migrations 000000..000003 **já aplicadas no banco real** `invtech-db` (schema criado,
  RLS FORCE confirmada). Antes disso, também validadas contra Postgres efêmero de teste.
- Suíte negativa **11/11 itens verde** (RLS isolation, WITH CHECK, append-only, scope 'all',
  CSRF 403, FK sector≠branch em asset/collaborator, dupla alocação ativa, MAIL_DRIVER=noop,
  FAIL-FAST abort, must_change_password gate, fail-closed sem SET LOCAL).
- Auth completo: Argon2id, sessions server-side revogáveis, auth_lookup/session_lookup na
  mesma transação (R6/R8), CSRF double-submit, middleware de tenant com Tx por requisição.
- `//go:embed` da UI (`web/embed.go`) servindo `web/index.html` original.
- Handlers dos 9 módulos + import idempotente — **cobertura de integração completa**:
  fluxo ponta-a-ponta (`business_flow_test.go`) + update/list/delete de cada módulo +
  criação de user/role via API (`module_crud_coverage_test.go`), ambos contra Postgres real.
- Seed (R10) — idempotente, testado com o binário real, bloqueado em produção pelo FAIL-FAST.
- Binário único `invtech-api` com modos `-migrate`, `-seed`, `-healthcheck` e serve padrão —
  todos testados via Docker (sem `go` instalado no host; toolchain usada via
  `golang:1.23-alpine` em container).

**Desvio registrado:** sqlc não foi usado — formalizado como emenda SQLC-01 pelo Mentor
(amendment 6), com as 3 condições cumpridas: registrado em ARQUITETURA.md, todo caminho de
query coberto por teste de integração, `queries/` (vazia) removida.

**Deploy gates (amendment 6):**
- G1 ✅ · G2 ✅ · G4 ✅ (evidências em PROGRESSO.md)
- G3 ⛔ **ainda bloqueado, mas agora por escopo, não por falta de MCP** — MCP-FORTCARGO foi
  autorizado (2026-07-23) e as tools `opsmonitor_*` funcionam, mas são só leitura. O registro em
  si (`ContainerPlaybook`/`HTTP_ENDPOINTS`) mora em `~/projetos/opsmonitor`, que vive no **srv100**
  — fora do escopo desta sessão de projeto (regra DFE: sessão de projeto só mexe no próprio
  projeto). Demanda atualizada com os dados concretos:
  `ai-workspace/Documentacao/governanca/demandas/opsmonitor-registrar-hermes-invtech.md`.
- G5 ✅ checklist de paridade dos 9 módulos em PROGRESSO.md.

**App `invtech-api` criada no Coolify (2026-07-23):** projeto `servicos-saas`, environment
`production`, uuid `j6yc0qzhslky713ra5qte0u4`, build pack `dockercompose` (usa o
`docker-compose.yaml` do repo), fonte GitHub via GitHub App `fortcargo-github` (repo privado
`Fortcargo/invtech`, branch `main`). As 17 env vars de `.env.example` foram definidas (incluindo
`DB_PASSWORD`/`DB_MIGRATE_PASSWORD` reais). **Deploy NÃO disparado ainda** (`instant_deploy=false`).

**Domínio `invtech.fortcargo.com.br` criado (2026-07-23):** consultei a skill `exposicao-segura`
antes — veredito: exposição direta é OK (InvTech tem login próprio com Argon2id/sessions/CSRF,
é o produto em si, não um painel de infra que precise de Basic Auth/Cloudflare Access na frente).
- DNS: `A invtech.fortcargo.com.br → 128.201.243.151` (mesmo padrão de `hermes`/`workflows`/
  `automation-engine` na zona — Hostinger, ttl 300).
- Coolify: `docker_compose_domains` vinculado ao serviço `invtech-api` →
  `https://invtech.fortcargo.com.br`. **Feito via PATCH direto na API do Coolify
  (`localhost:8000/api/v1/applications/...`), não pela tool MCP
  `coolify110_configurar_app`** — essa tool tem um bug real: o schema dela declara
  `docker_compose_domains` como array de *strings*, mas a API do Coolify exige array de objetos
  `{name, domain}`; toda tentativa pela tool MCP falhou com 422. Vale reportar pra quem mantém o
  `fortcargo-mcp` (não fiz — fora do escopo desta sessão de projeto).
- `fqdn` da app só reflete o domínio novo depois do próximo deploy (Traefik/labels são
  regenerados no build) — confirmado que o `docker_compose` já é o real (a correção do
  `docker-compose.yaml` do commit anterior já foi repuxada pelo Coolify, `expose: ['8080']`
  confirmado via API).
- **Deploy ainda não disparado** — domínio e env vars prontos, falta só decidir disparar.

**Topologia de git corrigida e validada contra os demais projetos (2026-07-23):** o remote
`servidor` apontava para `10.0.0.110` (mesmo host da produção — nenhuma redundância real).
Validei contra todos os projetos com working copy em `~/projetos` no srv110
(automation-engine, ai-workspace, fortcargo-backend, erp-fortcargo, hermes): **nenhum tem
remote `servidor`** — só `origin` (GitHub). O padrão real do srv110 é: clone de trabalho
apontando só pro GitHub; **srv100 é o único host que concentra os bare mirrors** de todos os
projetos (`~/repos/*.git`, dezenas de projetos confirmados lá).

Invtech agora segue exatamente esse padrão:
- Clone local no srv110 (`~/projetos/invtech`): só `origin` → `git@github.com:Fortcargo/invtech.git`.
- Bare mirror em `10.0.0.100:~/repos/invtech.git` — criado e sincronizado (histórico completo até
  este commit).
- Bare mirror órfão que eu mesmo tinha deixado em `10.0.0.110:~/repos/invtech.git` — **removido**,
  já que nenhum outro projeto tem mirror local no 110.

**Mirror automático criado (2026-07-23):** como nenhum mirror do 100 tinha sync automático
(bare repos sem remote fetch configurado, nenhum cron existente), criei um mecanismo dedicado
em vez de depender de push manual do 110:
- `10.0.0.100:~/repos/invtech.git` ganhou remote `origin` → `git@github.com:Fortcargo/invtech.git`
  com refspec de espelho (`+refs/heads/*:refs/heads/*` + tags).
- `10.0.0.100:~/scripts/mirror-invtech.sh` — `git fetch --prune` no mirror, com `flock` (evita
  sobreposição), log em `~/scripts/logs/mirror-invtech_cron.log`.
- Cron em 100: `*/30 * * * * ~/scripts/mirror-invtech.sh` — testado manualmente antes de agendar.

Assim o mirror em 100 fica sempre a no máximo 30min do GitHub, sem depender de ninguém lembrar
de dar push — puramente pull-based, desacoplado de onde o desenvolvimento acontece.

Também notei (fora do escopo desta sessão): `fortcargo-backend` e `erp-fortcargo` não têm mirror
em 100 ainda, e `hermes.git` existe órfão em 110 sem cópia em 100 — inconsistências do resto da
frota, não mexidas aqui.

**Deploy passou a usar GitHub Actions (2026-07-23)** — correção de uma leitura errada minha:
eu tinha dito que o Coolify deste host não tinha endpoint público, mas **tem sim**:
`coolify-fc.fortcargo.com.br` (rota Traefik `coolify.yaml`, gerada pelo próprio Coolify,
`http://coolify:8080` por trás, TLS letsencrypt) — só não aparecia nos labels Docker do
container `coolify` porque o Coolify gerencia essa rota via arquivo dinâmico do Traefik, não
via labels como um app comum. Confirmado pedindo pra eu vasculhar de novo.

Com esse endpoint confirmado, criei `.github/workflows/ci-deploy.yml`:
- **job `test`:** build + vet + testes unitários + sobe um Postgres 16-alpine como *service
  container* do próprio job + aplica migrations 000000..000003 + `bootstrap_db.sh` + roda a
  suíte negativa e os testes de integração completos (`-tags=integration`) — isso fecha a
  lacuna que eu tinha apontado antes (nada rodava automaticamente a cada push).
- **job `deploy`** (só roda se `test` passar): `curl` autenticado em
  `https://coolify-fc.fortcargo.com.br/api/v1/deploy?uuid=j6yc0qzhslky713ra5qte0u4` — endpoint
  confirmado com uma chamada de teste (uuid inválido → 404 "No resources found", prova que o
  endpoint existe sem disparar deploy real).
- Secret `COOLIFY_API_TOKEN` criado no repo (`Fortcargo/invtech`) — criptografado com a chave
  pública do GitHub Actions (libsodium sealed box) e enviado via API, reaproveitando o
  `COOLIFY_TOKEN_110` do cofre. **Recomendação:** trocar por um token do Coolify escopado só
  pra deploy (Settings → Keys & Tokens, permissão restrita), em vez do token de uso geral.

**Nota de segurança encontrada, fora do escopo desta sessão:** `coolify-fc.fortcargo.com.br`
expõe o **dashboard inteiro** do Coolify (login, API, terminal web), sem Basic Auth/Cloudflare
Access na frente — isso é o tipo de "plano de controle público" que a regra de exposição do
projeto proíbe. É uma condição pré-existente (não foi criada por mim), mas vale reportar —
não mexi porque trancar isso pode quebrar o próprio webhook que acabei de habilitar, e decidir
a forma correta de proteger (Cloudflare Access na frente, por exemplo) é uma decisão maior que
não me cabe tomar sozinho aqui.

## PRIMEIRO DEPLOY REAL — concluído (2026-07-23)

Push em `main` disparou `ci-deploy.yml`: job `test` verde (build/vet/unit/negativa/integração
contra Postgres real do job) → job `deploy` verde (`GET /api/v1/deploy` no Coolify). App subiu,
mas `/health` respondia 200 e `/health/ready` 503 (`pool.Ping()` falhando).

**Causa raiz:** build pack `dockercompose` do Coolify cria uma rede docker isolada por app
(`j6yc0qzhslky713ra5qte0u4`); `invtech-api` só estava nela, `invtech-db` está na rede `coolify`
— containers em redes diferentes não se enxergam, `DB_HOST` não resolvia.

**Fix:** declarado no próprio `docker-compose.yaml` — serviço `invtech-api` ganhou `networks:
[default, coolify]` + `networks.coolify.external: true` no topo do arquivo. Testado com um novo
push real: o container do deploy seguinte já subiu nas duas redes automaticamente, sem
intervenção manual — **é durável**, não precisa reconectar a cada deploy (diferente do que o
`CLAUDE.md` documenta para outros apps neste host; aqui funcionou de primeira, possivelmente por
ser um app novo criado já nesta versão do Coolify).

**Validado ao vivo:** `https://invtech.fortcargo.com.br/health` → 200 · `/health/ready` → 200
`{"db":"ok",...}` · `/` (UI embutida) → 200. Pipeline completo (push → CI → deploy → produção)
funcionando de ponta a ponta.

**Próxima ação exata:** G3 fica com a sessão `opsmonitor` via a demanda registrada — quando
resolvida, atualizar o registro com o container real (agora nomeado
`invtech-api-j6yc0qzhslky713ra5qte0u4-*`, muda a cada deploy — melhor monitorar pelo domínio
`invtech.fortcargo.com.br/health` do que por nome de container). Retornar ao Mentor com a
evidência de G1/G2/G4/G5 + primeiro deploy real bem-sucedido, conforme pedido em "next" da
amendment 6. Considerar trocar `COOLIFY_API_TOKEN` (secret do GitHub) por um token do Coolify
escopado só pra deploy, em vez do token de uso geral reaproveitado.

## Primeiro tenant real: Fortcargo (2026-07-23)

Criado via script ad-hoc de uso único (não commitado — `go run` direto contra o `invtech_db`
real, usando `internal/repository`/`pkg/security` pra manter Argon2id e RLS corretos, depois
apagado): tenant `Fortcargo` (slug `fortcargo`), plano `pro`, role `admin` com permissão total
nos 9 módulos, usuário `marcio.velten@fortcargo.com.br` (`must_change_password=true`, scope
`all`). Login testado de verdade contra a API em produção — 200, sessão emitida. Senha temporária
entregue fora deste arquivo (não versionar credencial real). Não usei o tenant `demo` do seed
(dados fictícios) — este é um tenant de verdade para uso real.

## Incidentes reais em produção (2026-07-23)

1. **Gateway Timeout** — `docker-compose.yaml` listava `networks: [default, coolify]` sem
   `networks.default` declarada no topo; o Compose criou uma 3ª rede ad-hoc que o `coolify-proxy`
   não alcança, e o Traefik às vezes roteava pra ela → site inteiro fora do ar. Mitigado ao vivo
   (disconnect da rede órfã) + corrigido na fonte (só `coolify` na lista).
2. **Vazamento de hash de senha + tela de usuários quebrada** — `repository.User` não tinha json
   tags: `GET /api/v1/users` serializava em PascalCase (`Email`, `RoleID`, `Status`) em vez de
   snake_case, e vazava `PasswordHash` (hash Argon2id) na resposta. A tela de usuários lia campos
   em snake_case e via tudo `undefined` — parecia que o cadastro de `rudson@rdsolutec.com.br`
   tinha falhado, mas ele **tinha sido criado com sucesso** (confirmado no banco antes do fix).
   Corrigido com json tags corretas + `json:"-"` em PasswordHash. A própria suíte de testes tinha
   um buraco (helper de teste não validava o shape de respostas em array) — corrigido também,
   com testes de regressão específicos pra isso.
3. **"Falha ao acessar o banco" ao cadastrar Empresa/Filial** — reportado por
   `rudson@rdsolutec.com.br` (print de tela, 20:43). Não era conectividade de banco: reproduzido
   direto no `invtech-db` (`docker exec ... psql -U invtech_app`, sob RLS, no tenant real dele) que
   um `INSERT` com CNPJ formatado (`11.222.333/0001-81`, 18 chars) falha com `22001` (`value too
   long for type character varying(14)`) — as colunas `company.cnpj`/`branch.cnpj` são
   `varchar(14)` (só dígitos), mas `web/index.html` não removia a máscara antes de enviar e o
   placeholder do campo (`Ex: 12.345.678/0001-90`) convidava o usuário a digitar formatado. Como
   `mapDBError` não tratava `22001`, caía no 500 genérico "falha ao acessar o banco" — mensagem
   enganosa, mascarando um erro de validação como se fosse infra. Corrigido nas duas pontas:
   frontend agora envia `.replace(/\D/g, '')` no CNPJ de empresa e filial;
   `internal/handlers/dberror.go` passou a mapear `22001` pra 400 "dados inválidos: valor excede o
   tamanho máximo permitido" em vez de 500 (defesa em profundidade — protege qualquer campo
   `varchar` de tamanho fixo, não só CNPJ). Validado: build/vet/testes unitários verdes + JS
   re-checado com `node --check`.

## Auditoria de CRUD dos 9 módulos (2026-07-23)

Pedido do usuário: validar se todas as rotas de CRUD funcionam de verdade e corrigir o que não
funcionar. Metodologia: tenant/usuário de QA descartável (criado via script ad-hoc de uso único,
apagado depois), bateria completa via curl simulando o navegador (create/list/update/delete em
produção real, não só nos testes) para os 9 módulos, cruzada com `routes.go` vs. toda chamada
`apiFetch` do frontend (sem mismatch de rota encontrado). Suíte de integração completa (unitários +
`-tags=integration`) rodada antes e depois de cada fix, contra Postgres real (owner sem BYPASSRLS).

**3 bugs reais encontrados e corrigidos:**
1. **`UpdateBranch`/`UpdateSector` devolviam o FK do pai zerado** (`company_id`/`branch_id` como
   `00000000-...`) na resposta da API — o dado no banco estava correto (não é corrupção), mas a
   struct de retorno era montada à mão em Go sem nunca ler `company_id`/`branch_id`, já que esses
   campos não são parâmetro do update (não são editáveis, corretamente). Corrigido com
   `RETURNING company_id`/`RETURNING branch_id` no próprio UPDATE.
2. **`UpdateAsset` devolvia `status` vazio** e **`UpdateUserRoleStatus` devolvia `email`/
   `auth_provider` vazios e `must_change_password` sempre `false`** — mesma causa raiz: struct de
   retorno montada só com os campos que o handler recebeu, não com o estado real pós-UPDATE.
   Corrigido com `RETURNING` nas colunas que o Go não tinha em mãos.
   **Nenhum teste pegava isso** — a suíte só validava o *status HTTP* do update, nunca o corpo da
   resposta; a checagem de shape só existia pro endpoint de *list* (que sempre relê do banco).
   Adicionadas asserções de regressão em `module_crud_coverage_test.go` pros 4 casos.
3. **XSS armazenado latente em `termo_html`** (resposta de `POST /allocations`): o backend
   concatenava `"<pre>" + nome_do_ativo + nome_do_colaborador + "</pre>"` sem escapar — nome de
   ativo/colaborador é texto livre do usuário. Hoje o frontend nunca renderiza `termo_html` via
   `innerHTML` (não é explorável pela UI atual), mas é um campo de API real, então qualquer uso
   futuro (ex: tela de impressão de termo) herdaria o XSS silenciosamente. Corrigido com
   `html.EscapeString` antes de concatenar — defesa em profundidade, não fix de comportamento.

Achado também durante esta auditoria (não é bug de CRUD, é o motivo de ter começado a
investigação): CNPJ com máscara quebrando cadastro de empresa/filial, já corrigido e documentado
acima em "Incidentes reais em produção", item 3.

## Code review via Codex CLI (2026-07-23)

Pedido do usuário: disparar o Codex (`codex exec -s read-only`, leitura completa do repo,
ARQUITETURA.md como referência de invariantes pretendidas) pra revisão de segurança e estrutura,
independente da minha própria auditoria de CRUD. 6 achados; triei cada um contra o código real
antes de agir (nem todo achado de IA se sustenta na verificação):

1. **Crítico — RBAC/escopo não é aplicado** (já conhecido, ver "Achado de escopo" acima).
   Confirmado, sem ação nova aqui — decisão de produto pendente (granularidade de permissão).
2. **Alto — XSS armazenado generalizado no frontend** (`web/index.html`, ~15 pontos): TODOS os
   `render*()` dos 9 módulos + os dropdowns + o termo de responsabilidade (`renderTermPreview`)
   montam HTML via template literal + `innerHTML` interpolando `nome`/`email`/`cargo`/`hostname`/
   `notes`/`cnpj`/`city`/`state`/`description`/`icon` sem escapar — um nome de ativo/colaborador
   como `<img src=x onerror=...>` persiste e executa no navegador de QUALQUER usuário que abra a
   tela (inclusive admin vendo/imprimindo o termo). Mais grave que o XSS que eu já tinha corrigido
   no backend (`termo_html`, que nem é usado pelo frontend hoje). **Corrigido**: helper `esc()`
   (escape de `&<>"'`) aplicado em todos os pontos de interpolação de dado do usuário — tabelas,
   dropdowns, termo de responsabilidade, toast de erro (`err.message` do backend). Validado com
   `node --check` após a mudança.
3. **Médio — `subscriptions`/`billing_events` sem RLS, com grants completos pro runtime**:
   confirmado direto no banco (`information_schema.role_table_grants`) — `invtech_app` tem
   SELECT/INSERT/UPDATE/DELETE nessas tabelas + `tenants`/`plans`, sem RLS. Risco é latente, não
   ativo: nenhum handler HTTP toca essas tabelas hoje (só `seed.go` e o teste de RLS). Não corrigido
   — é decisão arquitetural (aplicar RLS FORCE nessas tabelas ou revogar privilégios do runtime),
   registrando como pendência.
4. **Médio — alocação aceita ativo e colaborador de filiais diferentes**: o Codex alegou que
   "a arquitetura afirma" validação de filial no serviço — **não confirmei isso no ARQUITETURA.md**
   (não há menção). É uma lacuna de design real, mas não uma violação de spec documentada — pode
   ser intencional (ex: redistribuição de equipamento entre filiais). Não corrigido — decisão de
   produto, não bug de código.
5. **Médio — exclusão de usuário deixava `auth_lookup`/`session_lookup` órfãos**: confirmado real.
   `auth_lookup` não tem FK pra `users` (só pra `tenants`) — excluir um usuário sem limpar
   `auth_lookup` deixava o e-mail preso pra sempre (recriar batia em "registro duplicado" mesmo com
   o usuário original já apagado). **Verifiquei também a hipótese de sessão sobrevivendo à exclusão**
   (Codex não chegou a afirmar isso, mas eu suspeitei): `sessions` TEM `ON DELETE CASCADE` pra
   `users`, então isso já era seguro — confirmado revertendo o fix e rodando o teste, que falhou só
   no passo de recriar e-mail, não no de sessão. **Corrigido**: `DeleteUser` agora limpa
   `auth_lookup`/`session_lookup` na mesma transação. Teste de regressão adicionado
   (`module_crud_coverage_test.go`): cria usuário → loga → admin exclui → confirma 401 na sessão →
   recria com o mesmo e-mail → confirma 201 (antes do fix, dava 409). Confirmei que o teste pega a
   regressão revertendo o fix manualmente e rodando de novo (falhou como esperado).
6. **Baixo — senha de seed hardcoded e logada**: já mitigado pelo FAIL-FAST em produção; não
   corrigido agora (baixo risco, ambiente de dev apenas, fora do foco desta rodada).

## Gate de deploy: achado e decisão (2026-07-23)

Pergunta do usuário: o Coolify já dispara deploy sozinho via GitHub App a cada commit,
independente da Action? Confirmei via `application_deployment_queues` no Postgres interno do
Coolify (`docker exec coolify-db psql`) que **sim** — todo push em `main` gera DOIS deploys
distintos, em pares consecutivos:
1. `is_webhook=true` — o webhook nativo do Coolify (source `App\Models\GithubApp`), dispara em
   segundos, **sem esperar o job `test` do GitHub Actions**.
2. `is_api=true` — a chamada explícita do job `deploy` do `ci-deploy.yml`, ~1min depois, só após
   os testes passarem.

**Implicação real: o gate de CI é cosmético hoje.** Se um push tivesse código quebrado, o
webhook nativo já teria colocado em produção antes do job de teste terminar — o segundo deploy
só reimplanta o mesmo commit. Não existe campo (`is_auto_deploy_enabled` ou similar) na API nem
na tabela `applications` do Coolify pra desligar isso por app quando a fonte é `GithubApp` —
é inerente a esse tipo de conexão.

Tentei resolver via branch protection real no GitHub (required status check antes do merge) —
**bloqueado pelo plano**: a API recusou com 403 ("Upgrade to GitHub Pro or make this repository
public") porque `Fortcargo/invtech` é privado e a org está no plano Free (branch protection com
required checks/reviews em repo privado exige GitHub Team+).

**Decisão do usuário: aceitar o duplo-deploy por enquanto**, documentado aqui como risco
conhecido. Mantido no `ci-deploy.yml` o trigger `pull_request` (além de `push`) — o job `test`
roda em qualquer PR aberta manualmente contra `main` (disciplina de equipe, sem enforcement
automático do GitHub); o job `deploy` só roda em evento `push` (`if: github.event_name ==
'push'`), nunca em PR.

**Se isso precisar ser resolvido de verdade no futuro**, as duas opções tecnicamente viáveis
continuam registradas: (a) trocar a fonte git da app no Coolify de "GitHub App" pra "Repositório
público + deploy key manual" — remove o webhook nativo sem depender de plano do GitHub; (b)
assinar GitHub Team pra habilitar branch protection real em repo privado.

## RBAC: enforcement de role_permissions/user_scopes implementado (2026-07-23)

Pedido do orquestrador (pós-decisão do Mentor 132b005/D6), confirmado com o usuário: implementar
o enforcement, não só formalizar o gap. `role_permissions` e `user_scopes` já existiam no schema
e eram gravados desde o W4, mas nenhum middleware os lia — qualquer usuário autenticado (senha já
trocada) acessava e mutava qualquer um dos 9 módulos, sem respeitar role/escopo.

**Modelo implementado:**
- **Permissão (module/action)** — `role_permissions(role_id, module, action)`. Novo
  `middleware.RequirePermission(module, action)` aplicado em TODA rota de negócio em `routes.go`
  (exceto `/dashboard`, de propósito — é a rota que o frontend usa pra descobrir se a sessão está
  viva, tem que responder pra qualquer usuário autenticado independente de role). 403 se a role não
  tiver o par exato.
- **Escopo (empresa/filial)** — `user_scopes(user_id, scope_type ['all','company','branch'],
  scope_id)`. `repository.ResolveUserScope` expande escopo `'company'` pra incluir todas as
  filiais daquela empresa; escopo `'branch'` fica só na filial explícita (não sobe pra empresa —
  usuário restrito a uma filial não gerencia a empresa dona dela). Aplicado dentro de cada handler
  de **create/update/delete** dos módulos vinculados a empresa/filial (companies, branches,
  sectors, collaborators, assets, allocations), via `middleware.Scope(c)` +
  `requireCompanyScope`/`requireBranchScope`/`requireUnrestrictedScope` (novo
  `internal/handlers/authz.go`). Criar uma empresa nova exige escopo `'all'` — não há empresa "dona"
  pra checar contra o escopo de um usuário restrito.
- **Módulos NÃO escopados por empresa/filial** (só permissão, de propósito): `asset_types`
  (catálogo tenant-wide), `users`/`roles` (administração do tenant), `import` (payload heterogêneo,
  validar escopo por linha seria bem mais caro — fica pendência se algum dia for pedido).
- **Leitura (list/get) NÃO é filtrada por escopo nesta rodada** — `GET /companies`,
  `/assets` etc. continuam devolvendo todos os registros do tenant pra qualquer usuário com a
  permissão de módulo, independente do escopo dele. Decisão consciente por tempo/escopo: a
  prioridade foi fechar o buraco mais grave (usuário restrito **alterando/apagando** dado fora do
  seu escopo), não o de visibilidade em leitura (exposição dentro do MESMO tenant, severidade bem
  menor que vazamento entre tenants). Filtrar list/get por escopo fica registrado como pendência
  real se precisar de verdade.

**Achado crítico durante a implementação:** as duas roles reais em produção (`admin` de
`marcio.velten@fortcargo.com.br`, `KeyUser` de `rudson@rdsolutec.com.br`, ambas no tenant
Fortcargo) **já tinham as 44 permissões completas** (11 módulos × 4 ações) e `scope_type='all'` —
verificado direto no banco antes de aplicar o enforcement, porque se não tivessem, os dois únicos
usuários reais do sistema ficariam bloqueados de tudo assim que isso fosse ao ar. Não precisou de
nenhuma correção de dado em produção.

**Achado de robustez no frontend, causado pela própria mudança:** `loadAllData()` usava
`Promise.all` sem tratamento de erro — um 403 em QUALQUER módulo (agora uma resposta válida e
esperada pra role restrita) rejeitava a promise inteira e a tela ficava em branco, mesmo em
módulos que o usuário TEM acesso. Isso inutilizaria o RBAC na prática assim que a primeira role
restrita fosse criada. Corrigido com `fetchListOrEmpty()` — 403 vira lista vazia (silencioso),
outros erros continuam logados no console.

**Fixtures de teste ajustados:** `business_flow_test.go`/`module_crud_coverage_test.go` criavam a
role "admin" de teste via SQL puro, sem `role_permissions`/`user_scopes` — todo teste começou a
tomar 403. Corrigido com `repository.FullPermissionSet()` (novo, também usado potencialmente por
outros fixtures) + `scope_type='all'` explícito.

**Teste de regressão dedicado:** `internal/app/authz_test.go` —
`TestAuthz_PermissionAndScopeEnforcement` — usuário com role restrita a só "assets" e escopo só à
Filial A: confirma 403 em módulo sem permissão (`/companies`), 403 ao tentar criar ativo na Filial
B (fora do escopo), e sucesso completo (create/delete) na Filial A (dentro do escopo).

Validado: build/vet + suíte completa (unitários + integração, incluindo o teste novo) verde.

## Invariantes
- INV-1: exatamente UM executor invtech ✅ (só a sessão systemd).
- INV-2: provado VAZIO antes do W3 ✅.
- INV-3: nenhuma migration antes do W1 (commitado) — W2/W3 não criaram migrations; W4 criou
  000000..000003, testadas contra Postgres efêmero E aplicadas no banco real do Coolify.

## Referências
- ARQUITETURA.md (raiz, § Emenda SQLC-01) · docs/W3-gate-decision.md · MentorDecision
  dfe-invtech-saas-etapa0-2026-07-19 (amendment 6)
