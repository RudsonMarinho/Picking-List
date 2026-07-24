# AGENTS.md — Protocolo de Handoff entre IAs
## Projeto: [NOME DO PROJETO]
## Fortcargo Transportes LTDA · Dev Factory Elite v2.1

> **Criado em:** [DATA]
> **Stack:** [Go 1.23 / Python 3.12 / Híbrido]
> **Banco:** [PostgreSQL `10.0.0.100:5432` banco `[NOME_BANCO]` / SQL Server `10.0.0.17:1433`]
> **Repositório:** `git@github.com:marciovelten/[PROJETO].git`
> **Servidor:** `/home/marcio/projetos/[PROJETO]/`

---

## Para qualquer IA que abrir este projeto

Leia nesta ordem antes de fazer qualquer coisa:

```
1. AGENTS.md       ← este arquivo — entenda o contexto geral
2. SESSAO.md       ← estado atual e próximo passo EXATO
3. HANDOFF.md      ← o que foi feito e o que falta
4. ARQUITETURA.md  ← decisões técnicas e estrutura do projeto
```

Se `SESSAO.md` descrever trabalho incompleto de outra IA, **continue de onde parou** — não reinicie.

---

## Mapa de responsabilidades — Dev Factory Elite

| Etapa | IA | Papel | Status |
|-------|-----|-------|--------|
| 0 | **Claude.ai (Mentor)** | Validação do briefing e decisões arquiteturais | [ ] |
| 1 | **Gemini** | Geração do ARQUITETURA.md completo | [ ] |
| 2 | **Claude.ai (Mentor)** | Validação da arquitetura contra infra real | [ ] |
| 3 | **Antigravity** | Scaffold — estrutura de pastas, Dockerfile, configs | [ ] |
| 4 | **Claude Code** | Implementação da lógica de negócio e integrações | [ ] |
| 5 | **Codex** | Code review, race conditions, edge cases | [ ] |
| 6 | **Gemini** | Workflow n8n (se aplicável) | [ ] |
| 7 | **—** | Deploy via Coolify | [ ] |

> Atualizar o status de cada etapa ao concluir: `[ ]` → `[x]`

---

## Estado atual do projeto

```
Etapa em andamento : [NÚMERO] — [NOME DA ETAPA]
IA responsável     : [NOME DA IA]
Última atualização : [DATA E HORA]
```

**O que foi implementado até agora:**
- [ ] [descrever módulo/feature 1]
- [ ] [descrever módulo/feature 2]

**O que falta implementar:**
- [ ] [próximo módulo]
- [ ] [próxima feature]

---

## Protocolo de handoff — quando uma IA fica sem crédito

### Passo 1 — antes de parar

```bash
# Commitar estado atual (mesmo incompleto)
git add -A
git commit -m "wip: [o que estava fazendo] — handoff para [IA substituta]"
git push origin main && git push servidor main
```

### Passo 2 — atualizar SESSAO.md com precisão

```
- O que foi feito nesta sessão
- Arquivos criados ou modificados (listar todos)
- Próximo passo EXATO (arquivo, função, endpoint)
- Qual arquivo abrir primeiro na próxima sessão
- Qual IA substituir e qual comando usar para iniciar
```

### Passo 3 — abrir a IA substituta com

```
Arquivos obrigatórios para colar:
1. AGENTS.md   ← contexto geral do projeto e estado das etapas
2. SESSAO.md   ← o que fazer agora
3. CLAUDE.md   ← contexto técnico do projeto (pasta .claude/)
```

---

## Matriz de substituição — quem assume quando

| IA que parou | Substituta direta | Como acionar |
|---|---|---|
| **Claude Code** | Antigravity (VS Code) | Abrir pasta no VS Code → colar SESSAO.md no chat |
| **Claude Code** | Codex (task isolada) | Abrir Custom GPT → colar SESSAO.md + trecho de código |
| **Claude Code** | Gemini CLI | `gemini` na pasta do projeto → colar SESSAO.md |
| **Antigravity** | Claude Code | `cd [PROJETO] && claude` → colar SESSAO.md |
| **Codex** | Claude Code | `cd [PROJETO] && claude` → pedir continuação do review |
| **Gemini** | Claude.ai (Mentor) | Project Mentor Técnico → colar ARQUITETURA.md |
| **Claude.ai Mentor** | ChatGPT Pro | Colar AGENTS.md + SESSAO.md + ARQUITETURA.md manualmente |

---

## Regras que toda IA deve respeitar neste projeto

### Proibições absolutas

| Proibição | Motivo |
|---|---|
| `host.docker.internal` | Não resolve no Ubuntu — usar `10.0.1.1` |
| `:latest` em imagens Docker | Breaking changes silenciosos em produção |
| `DATABASE_URL` com `@` na string | Quebra com senha contendo `@` ou `%` |
| ORM em Go | Sempre pgx v5 nativo + sqlc |
| `create_all()` em Python | Sempre Alembic com versioned migrations |
| Hardcode de senhas ou API keys | Sempre `.env` + config |
| Banco de outro projeto | Banco isolado — nunca BD_global |

### Stack deste projeto

```
[PREENCHER COM A STACK ESPECÍFICA DO PROJETO]
Exemplo Go:
  Framework: Fiber v2
  Banco: pgx v5 + sqlc + golang-migrate
  Config: viper | Logs: zerolog | Cache: go-redis v9

Exemplo Python:
  Framework: FastAPI + uvicorn
  Banco: SQLAlchemy async + Alembic + URL.create()
  Config: pydantic-settings | Cache: redis.asyncio
```

### Infraestrutura deste projeto

```
Servidor:     10.0.0.100
Coolify:      http://10.0.0.100:8000
Banco:        [endereço e nome do banco isolado]
Redis:        [host:porta interno]
Porta app:    [número — verificado com check-ports.sh]
URL pública:  [https://[PROJETO].ecrcargas.com.br]
```

---

## Protocolo de sincronização ao fechar qualquer etapa

**Camada 1 — sempre:**
```bash
# Atualizar os três arquivos
PROGRESSO.md + SESSAO.md + HANDOFF.md

# Commit separado de docs
git add -A
git commit -m "docs: sync etapa [N] — [resumo do que mudou]"
git push origin main && git push servidor main
```

**Camada 2 — conforme o que mudou:**
```
Novo endpoint       → README.md + ARQUITETURA.md
Nova integração     → AGENTS.md + GEMINI.md + .claude/CLAUDE.md
Nova variável       → .env.example + ARQUITETURA.md
Novo container      → ESTRUTURA-SERVIDOR.md (após deploy)
```

> ⚠️ Uma etapa **não está concluída** enquanto os documentos não refletirem o estado real.
> A próxima IA começa do estado descrito nos docs — não do código.

---

## Histórico de sessões

| Data | IA | Etapa | O que foi feito |
|------|----|-------|-----------------|
| [DATA] | Antigravity | 3 | Scaffold inicial — estrutura criada |
| | | | |
