# InvTech — Gestão de Ativos e Recursos de TI (SaaS · Fase 1)

Sistema multi-tenant de gestão de ativos de TI. Etapa 0 do DFE.

- **Stack:** Go 1.23 + Fiber v2 + pgx v5 + golang-migrate
- **Banco:** PostgreSQL isolado `invtech_db` (RLS FORCE, multi-tenant)
- **Servidor:** srv110 (10.0.0.110) · Coolify grupo `servicos-saas`
- **Frontend Fase 1:** `//go:embed` do HTML em `web/index.html` (sem Vercel)
- **Error ID prefix:** `INV` · **Senhas:** Argon2id · **Sessão:** server-side revogável

> **Desvio de stack (SQLC-01):** a decisão original previa sqlc para geração
> de queries. Formalizado pelo Mentor (amendment 6) como emenda válida só
> para a Fase 1 do invtech: queries pgx parametrizadas escritas à mão em
> `internal/repository/*.go`. Detalhe e condições cumpridas em
> ARQUITETURA.md § Emenda SQLC-01.

## Estado

**W4 implementado, validado e migrado no banco real** (migrations, FAIL-FAST,
RLS, Argon2id, sessions, CSRF, os 9 módulos + import idempotente, seed).
Suíte negativa completa (11/11 itens de ARQUITETURA.md) verde, com cobertura
de integração de todo caminho de query contra Postgres real (fluxo
ponta-a-ponta + update/list/delete de cada módulo).

`invtech-db` já existe no Coolify (`servicos-saas`), migrations aplicadas,
role `invtech_app` com senha real e validado em produção (G1/G2 da
MentorDecision amendment 6 — evidência em PROGRESSO.md). Ainda não existe a
aplicação `invtech-api` deployada — só o banco. G3 (registro no OpsMonitor)
segue bloqueado nesta sessão por falta de MCP autorizado.

## Documentação

- `ARQUITETURA.md` — decisões, schema, contrato de API, suíte negativa (fonte da verdade)
- `SELF-HEALER-INTEGRACAO.md` — contrato de monitoramento (OpsMonitor/Sentinela)
- `ORQUESTRADOR.md` — regras de orquestração + playbook de erro crítico
- `.claude/CLAUDE.md` — contexto do projeto para o Claude Code

## Ordem de implementação (W4)

1. FAIL-FAST de boot (antes de qualquer migration)
2. Migrations 000000..000003
3. Suíte negativa VERDE
4. `//go:embed` da UI + Argon2id + sessions + lookups + CSRF
5. Handlers dos 9 módulos + import idempotente
6. Seed
