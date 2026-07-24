# ORQUESTRADOR.md — [NOME DO PROJETO]
# DFE v4.2.0 | Playbook de erro crítico e recuperação (legível por agente)

> Preencher ao criar o projeto. Este arquivo é lido pelo Sentinela/OpsMonitor e pela sessão dona
> para decidir o que reiniciar, o que limpar, e quando parar e chamar humano.

## QUANDO ACIONAR MANUS NESTE PROJETO

Acionar sempre que precisar **aprofundar conhecimento** antes de implementar — não é subagente do
Claude Code, é um agente externo com web browsing, acionado via webhook (nenhuma API key local):

- Regulação específica do domínio do projeto (LGPD, setorial, etc.) antes de implementar
- Documentação de API externa não documentada localmente
- Comparação de abordagens de mercado antes de decidir uma feature nova
- Benchmark de biblioteca/serviço antes de adotar

```bash
curl -X POST https://n8n.ecrcargas.com.br/webhook/manus-dispatch \
  -H "Content-Type: application/json" \
  -d '{"tipo":"pesquisa","projeto":"[NOME DO PROJETO]","tema":"<tema>","arquivo_destino":"<arquivo>","prompt_manus":"<prompt>"}'
```

Usar `\n` escapado (aspas simples no bash) para quebras de linha no `prompt_manus` — nunca quebra
de linha crua dentro da string JSON. Pipeline: webhook → cria task na Manus → consulta status →
notifica WhatsApp `dfe-alerts` com o `task_id`. Workflow n8n: `manus-intelligence-dispatcher` (ativo).
Detalhe completo: `ai-workspace/Documentacao/apis/Guia de Configuração da API do Manus no n8n.md`.

## Critérios de ERRO CRÍTICO (parar e notificar — classe C)
- [ ] <ex.: migration falha por constraint 2×>
- [ ] <ex.: /health/ready 503 por > N min após redeploy>
- [ ] <ex.: credencial/segredo exposto em commit pusheado>

## O que NÃO é crítico (tentar resolver — classe A/B)
- [ ] <ex.: container unhealthy 1 ciclo → aguardar próximo probe>
- [ ] <ex.: fila crescendo → escalar worker via Coolify (A)>
- [ ] <ex.: vazamento de recurso no código → abrir tarefa no SESSAO.md (B)>

## Playbook de recuperação (ações seguras — classe A)
| Sintoma | Ação segura | Como |
|---|---|---|
| container unhealthy 2+ ciclos | redeploy | Coolify API (nunca docker manual) |
| processo/recurso vazando | restart + limpar | Coolify + limpeza idempotente |
| token/cert expirando | renovar | <comando> |

## Quando PARAR e chamar o Marcio (human-in-the-loop)
- Ação destrutiva/ambígua, decisão de negócio, ou falha que não resolve em N tentativas (circuit-breaker).
- Notificar via WhatsApp `dfe-alerts` (Evolution API).

## Limites de recurso do serviço
- `mem_limit`: <valor>  ·  `pids_limit`: <valor>  ·  cleanup garantido em: <finally/async-with/lifespan>
