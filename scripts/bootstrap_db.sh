#!/usr/bin/env bash
#
# scripts/bootstrap_db.sh
#
# C3: completa o provisionamento do role de runtime (invtech_app) que a
# migration 000000_create_role não pode fazer — golang-migrate não
# interpola variáveis, e a senha NUNCA fica versionada em migration.
#
# Pré-requisito: migration 000000_create_role já aplicada (o role precisa
# existir antes deste script rodar).
#
# Roda com a credencial de OWNER (DB_MIGRATE_USER) — NUNCA com invtech_app.
# P1 (MentorDecision dfe-invtech-saas-etapa0-2026-07-19, amendment 5):
# DB_USER e DB_MIGRATE_USER são obrigatoriamente roles distintos.
#
# Variáveis de ambiente exigidas:
#   DB_HOST, DB_PORT, DB_NAME
#   DB_MIGRATE_USER, DB_MIGRATE_PASSWORD   (owner — conecta e executa)
#   DB_USER, DB_PASSWORD                   (invtech_app — recebe senha + GRANT CONNECT)
#
# Idempotente: pode rodar em todo deploy sem efeito colateral.

set -euo pipefail

: "${DB_HOST:?DB_HOST não definido}"
: "${DB_PORT:?DB_PORT não definido}"
: "${DB_NAME:?DB_NAME não definido}"
: "${DB_MIGRATE_USER:?DB_MIGRATE_USER não definido}"
: "${DB_MIGRATE_PASSWORD:?DB_MIGRATE_PASSWORD não definido}"
: "${DB_USER:?DB_USER não definido}"
: "${DB_PASSWORD:?DB_PASSWORD não definido}"

if [ "$DB_USER" = "$DB_MIGRATE_USER" ]; then
  echo "ERRO: DB_USER e DB_MIGRATE_USER não podem ser o mesmo role — anula RLS e least privilege (P1)." >&2
  exit 1
fi

PGPASSWORD="$DB_MIGRATE_PASSWORD" psql \
  -v ON_ERROR_STOP=1 \
  -h "$DB_HOST" -p "$DB_PORT" -U "$DB_MIGRATE_USER" -d "$DB_NAME" \
  -v app_user="$DB_USER" \
  -v app_password="$DB_PASSWORD" \
  -v db_name="$DB_NAME" \
  <<'SQL'
ALTER ROLE :"app_user" WITH PASSWORD :'app_password';
GRANT CONNECT ON DATABASE :"db_name" TO :"app_user";
SQL

echo "bootstrap_db.sh: role $DB_USER pronto (senha definida, CONNECT concedido em $DB_NAME)."
