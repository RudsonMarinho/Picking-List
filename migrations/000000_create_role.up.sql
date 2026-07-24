CREATE EXTENSION IF NOT EXISTS "citext";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE ROLE invtech_app WITH LOGIN
  NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS;
-- Senha definida no provisionamento (Coolify) via scripts/bootstrap_db.sh,
-- nunca em migration.

GRANT USAGE ON SCHEMA public TO invtech_app;

-- C2: alcança TODAS as tabelas futuras, inclusive 000004+ (Fase 2).
ALTER DEFAULT PRIVILEGES IN SCHEMA public
  GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO invtech_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
  GRANT USAGE, SELECT ON SEQUENCES TO invtech_app;
