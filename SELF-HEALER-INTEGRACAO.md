# Contrato de Monitoramento & Auto-correção — OBRIGATÓRIO para todo novo projeto
# DFE v4.2.0 | Fortcargo | Atualizado: 2026-07-04

> Todo serviço do ecossistema **nasce** cumprindo este contrato, para ser monitorado e
> auto-corrigido pelo **OpsMonitor → Sentinela** (malha triangular 100/110/VPS).
> Fonte da decisão: `ai-workspace/Documentacao/governanca/BRIEFING_MENTOR_SENTINELA_TRIANGULAR.md` (seção 6).
> **Regra de ouro:** problema silencioso que pode derrubar serviço/servidor NÃO pode existir sem ser
> detectado e tratado automaticamente.

---

## As 8 obrigações do contrato (o projeto já nasce com elas)

1. **`GET /health` rico** — não basta `{"status":"ok"}`. Expor liveness **e** readiness **e** auto-métricas de risco:
   nº de processos filhos/browsers abertos, RSS, conexões abertas, tamanho de fila, idade do último job.
   - Liveness barato (`/health` → 200) para o Coolify/probe. Readiness/dependências (ping de banco etc.)
     em **`/health/ready`** — para não "flapar" o self-healer se uma dependência cair por segundos.
2. **Endpoints self-healer** (se houver fluxo multi-step / interativo) — `/admin/...` para inspeção e ação
   controlada (ver seção "Endpoints self-healer" abaixo).
3. **Limites de recurso no deploy** — Docker `mem_limit`, `pids_limit`, `ulimit` no compose (contêm vazamento
   antes de derrubar o host). Já vêm no `docker-compose.yaml` do template.
4. **Cleanup garantido** — todo recurso externo (browser Playwright, conexão, subprocesso, arquivo temporário)
   fechado em `finally` / `async with` / `lifespan shutdown`. (O incidente dos 512 Chromium zumbis nasceu daqui.)
5. **`ORQUESTRADOR.md`** — critérios de erro crítico + playbook de recuperação legível por agente
   (o que reiniciar, o que limpar, quando parar e chamar humano). Stub já vem no template.
6. **Teste de regressão por incidente conhecido** — ex.: "N automações → 0 chromium órfãos".
7. **Manifesto de sinais** — `docs/observabilidade.md`: quais métricas o serviço expõe, limiares, e a ação
   segura correspondente (mapa **sinal → correção classe A/B/C**). Stub já vem no template.
8. **Versionado no DFE** — header `DFE v4.2.0`, no radar do `dfe-sync`.

---

## Classes de auto-correção (como o Sentinela decide agir)

| Classe | O que é | Quem executa |
|---|---|---|
| **A — auto-segura** | reversível: restart via **Coolify** (nunca docker manual), limpeza de zumbis/temp, rotação de log, renovação de token/cert, reset de sessão | o próprio Sentinela, com backup antes + idempotência + rate-limit + registro |
| **B — código** | correção que mexe em código | **NUNCA automático** → vira **tarefa no `SESSAO.md`** do projeto, para a sessão dona corrigir com `code-reviewer` + testes |
| **C — destrutiva/ambígua** | risco alto ou incerto | **para e notifica o Marcio** (WhatsApp `dfe-alerts`) — human-in-the-loop |

O `docs/observabilidade.md` do projeto deve classificar **cada sinal** em A/B/C.

---

## Endpoints self-healer (para fluxos multi-step / interativos)

```
GET  /health                → {"status":"ok"}                       (liveness, 200)
GET  /health/ready          → readiness + auto-métricas de risco    (503 se não-pronto)
GET  /admin/sessions/stuck?threshold_minutes=15   → [ {session_id, current_step, stuck_minutes, is_paused, ...} ]
GET  /admin/sessions/current-question?phone=...   → {question_text, step_key}   (para re-envio)
POST /admin/sessions/send-pause-prompt            → {message, session_id}       (prompt de pausa gentil)
```
O healer chama `stuck` a cada 60s; usa `send-pause-prompt` para diferenciar inatividade humana de falha técnica.

---

## Registrar no OpsMonitor (obrigatório após o 1º deploy)

1. Mapear o container prefix no `ContainerPlaybook` (`internal/healer/playbook_container.go`):
   ```go
   containerAppMap: map[string]string{
       "novo-servico-prefix": "uuid-coolify-do-novo-servico",
   },
   ```
2. Adicionar o serviço nos `HTTP_ENDPOINTS` do OpsMonitor (`.env`) para o probe externo.
3. Confirmar que o Sentinela passa a ver o `/health` do serviço.

---

## Ciclo do healer para um serviço integrado

```
Container unhealthy (2+ ciclos)  → ContainerPlaybook → redeploy Coolify → alerta WA
Sessão travada + container unhealthy → GET /current-question → re-envia via Evolution API → alerta WA
Sessão travada + container saudável  → POST /send-pause-prompt (inactivity) → aguarda colaborador
                                       → sem resposta → (resume) → após N lembretes → alerta WA "INTERVENÇÃO HUMANA"
Falha desconhecida               → Claude Agent → analisa logs + contexto → age via API → alerta WA
```

---

## Checklist de integração (novo projeto)

- [ ] `GET /health` → 200 `{"status":"ok"}` + `GET /health/ready` (readiness + métricas de risco)
- [ ] Dockerfile com `HEALTHCHECK` para `/health` (sem `curl` em distroless — usar binário próprio ou `wget`)
- [ ] `mem_limit` / `pids_limit` no `docker-compose.yaml`
- [ ] Cleanup de todo recurso externo em `finally`/`async with`/`lifespan`
- [ ] `ORQUESTRADOR.md` preenchido (erro crítico + playbook)
- [ ] `docs/observabilidade.md` preenchido (mapa sinal → ação A/B/C)
- [ ] Teste de regressão do(s) incidente(s) previsível(is)
- [ ] Se fluxo multi-step/interativo: endpoints `/admin/...`
- [ ] Container prefix mapeado no `ContainerPlaybook` do OpsMonitor + `HTTP_ENDPOINTS` no `.env`
- [ ] Header `DFE v4.2.0` (radar do `dfe-sync`)
