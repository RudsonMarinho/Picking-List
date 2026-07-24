# Segredos — regra do projeto (nunca expor)

> Herdado do `_template`. Vale para todo projeto novo do ecossistema DFE/Fortcargo.

## Regra

**Nenhum segredo em texto puro no repositório** — nunca em código, script, doc, `.env`
commitado, compose ou log. Isso inclui: senhas, API keys, tokens, chaves privadas,
strings de conexão com senha embutida.

## Como fazer certo

- **Runtime (scripts/serviços):** o valor mora só no cofre `~/infra-credenciais.env`
  (chmod 600, gitignored). O script faz `source /home/marcio/infra-credenciais.env` e usa
  `"$NOME_DA_VAR"`. Ex.: `EVO_KEY="${EVOLUTION_API_KEY:?ausente no cofre}"`.
- **Container/Coolify:** injetar via env do Coolify — nunca literal no compose/código.
- **Documentação:** usar placeholder `{{cofre:NOME_DA_VAR}}` no lugar do valor.
- **`.env`:** só `.env.example` com placeholders (`CHANGE_ME`, `<sua-chave>`). O `.env`
  real é gitignored.

## Gate automático (pre-commit)

Este template traz `.githooks/pre-commit` que barra commit com segredo em texto puro.
**Ativar no projeto (uma vez):**

```bash
git config core.hooksPath .githooks
```

Falso positivo pontual: `git commit --no-verify` (com consciência). Ajuste os padrões
em `.githooks/pre-commit` se o projeto tiver um caso legítimo recorrente.

## Se um segredo vazou

1. Rotacionar o valor (trocar no sistema de origem) — remover do histórico não basta,
   considere-o comprometido.
2. Atualizar o cofre com o novo valor.
3. Sanear o repo (placeholder/var) e, se já foi pusheado, avaliar `git filter-repo`.

Referência da remediação-modelo: `~/documentacao/seguranca/remediacao-segredos-scripts-2026-07-21.md`.
