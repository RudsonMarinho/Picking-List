# W3 — Gate Foundry (PILOT-FOUNDRY-01) — Decisão registrada

> ⚠️ **CORREÇÃO (2026-07-23, Mentor `132b005`, caso C2):** o `agent_creation_gate.py` **não rodou**.
> `NO_AGENT_REQUIRED → sessionFallback` foi **conclusão HUMANA** (árvore da MentorDecision aplicada à mão).
> Este piloto **não é evidência sobre o Foundry** — prova só que existe uma sessão dedicada governada;
> `materialize_real()` segue `NotImplementedError`. Registro canônico:
> `ai-workspace/Documentacao/governanca/INVTECH_W3_GATE_RESULT_2026-07-23.md`.

- **Data/hora:** 2026-07-23 01:17:50 -03 (start da sessão dedicada)
- **Executor do gate:** orchestrator@srv100 (chefe)
- **Gate:** PILOT-FOUNDRY-01 (checkpoint 2 do Foundry)
- **Input:** capabilityType=projeto · project=invtech · node=srv110

## Precondition — INV-2 (provado)
`systemctl --user list-units/list-unit-files | grep invtech` no srv110 = **VAZIO** antes do start.
Nenhum executor invtech (sessão ou agente) existia. Evidência colhida 2026-07-23.

## Decisão (conclusão HUMANA — o gate NÃO rodou como código): sessionFallback
Desfecho **VÁLIDO** do piloto (o dispatch proíbe forçar CREATE). O modelo canônico DFE de executor
de projeto é a sessão systemd `claude-remote@<projeto>` (ver mapa de sessões). Não há necessidade de
materializar um agente novo — a sessão systemd É o executor. 0 credencial nova, 0 permissão nova.

## Execução (sessionFallback)
1. `trust-claude-dir.sh ~/projetos/invtech` — pasta trusted (backup do ~/.claude.json feito).
2. Template `claude-remote@.service` (srv110): inserido `invtech) DIR=/home/marcio/projetos/invtech`
   ANTES do `*)` — evita queda em `~` e colisão de bridge-pointer com `serverfortcargo`. Backup em
   `~/backups-config/`. `daemon-reload`.
3. `systemctl --user start + enable claude-remote@invtech`.
   PROIBIDO (e não usado): `tmux-bootstrap.sh`, `start-claude-remote-all.sh`.

## Validação
- Serviço `active (running)`, `enabled`. Main PID python3 → filho `claude remote-control --name invtech`.
- **cwd do processo = `/home/marcio/projetos/invtech`** (DIR correto, não `~`).
- **INV-1:** exatamente 1 executor invtech ativo (só a sessão systemd; nenhum agente).
- **0 tarefa de produção:** sessão ociosa, aguardando abertura no app + liberação do Mentor (W4).

## Retorno
W3 **retorna ao Mentor** antes do W4 (implementação). Registro do piloto em `LICOES_APRENDIDAS.md`
é trabalho não-bloqueante da sessão ai-workspace.
