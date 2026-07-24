# CLAUDE.md — InvTech (contexto do projeto)

Sistema SaaS multi-tenant de Gestão de Ativos de TI. Etapa 0 do DFE.
Decisão vigente: MentorDecision `dfe-invtech-saas-etapa0-2026-07-19` amendment 4.

## Onde roda
- Nó: **srv110** (10.0.0.110) · gateway Docker 10.0.0.110
- Coolify grupo `servicos-saas` · banco isolado `invtech_db`
- Repo: `Fortcargo/invtech` · Error ID prefix: `INV` · Classe de autonomia: **B**

## Regras absolutas (herda global + específicas)
- Go: nunca ORM — sempre pgx v5 + sqlc
- Banco: nunca `DATABASE_URL` com `@` — `pgx.ParseConfig()`. Nunca `create_all` — golang-migrate.
- Multi-tenant: FORCE RLS em toda tabela de negócio; `SET LOCAL app.tenant_id` (nunca `SET`).
- Senhas Argon2id. Sessão server-side revogável. CSRF double-submit em toda mutação.
- FAIL-FAST no boot em produção (ver ARQUITETURA § Segurança).
- Docker: nunca `:latest`. API via `expose` (não `ports`) — Traefik alcança pela rede.
- Nunca filtrar por IP (SNAT do pfSense).

## Antes de implementar (W4)
Ler nesta ordem: `ARQUITETURA.md` (fonte da verdade) → `SESSAO.md` → `PROGRESSO.md`.
Seguir a "Ordem de implementação" do ARQUITETURA. Suíte negativa VERDE antes de qualquer handler.
